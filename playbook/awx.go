package playbook

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/alvnukov/awx-go"
	"github.com/alvnukov/awx-go/playbook/model"
)

func (a *Playbook) Run(ctx context.Context, run model.Run) error {

	logrus.WithFields(logrus.Fields{
		"InventoryName": run.InventoryName,
		"TemplateName":  run.TemplateName,
		"Timeout":       run.Timeout,
		"Hosts":         run.Hosts,
		"Vars":          run.Vars,
	}).Info("Запускаю выполнение роли Playbook")

	if run.InventoryName == "" {
		return fmt.Errorf("awx: no name specified")
	}

	if run.TemplateName == "" {
		return fmt.Errorf("awx: no template name specified")
	}

	if len(run.HostList()) == 0 {
		return fmt.Errorf("awx: no hosts specified")
	}

	// Обновление аттрибутов хоста в Inventory
	inventory, err := a.ResetInventory(ctx, run)
	if err != nil {
		return fmt.Errorf("error occured on ResetHost: %w", err)
	}

	templatesOpts := map[string]string{"name": run.TemplateName}
	templates, err := a.client.JobTemplateService.ListJobTemplates(ctx, templatesOpts)
	if err != nil {
		return fmt.Errorf("error occured on list job templates: %w", err)
	}

	template, ok := templates.GetByName(run.TemplateName)
	if !ok {
		return fmt.Errorf("template '%s' not found", run.TemplateName)
	}

	// Подготовка конфигурации для запуска Playbook'a
	templateArguments := map[string]interface{}{
		"inventory":  inventory.ID,
		"limit":      strings.Join(run.HostList(), ","),
		"extra_vars": JSONUnmarshalString(run.Vars),
	}

	// Запуск Playbook'a
	job, err := a.client.JobTemplateService.Launch(ctx, template.ID, templateArguments)
	if err != nil {
		return fmt.Errorf("error occured on job launch: %w", err)
	}

	if err := awx.WaitForSuccessJobFinish(a.client, job.Job, run.Timeout); err != nil {
		return fmt.Errorf("success status waiting for job '%d' failed, err: %w", job.Job, err)
	}

	return nil
}

func (a *Playbook) prepareInventory(ctx context.Context, inventoryName string) (*awx.Inventory, error) {

	// 1) Проверяем наличае Inventory, если нет - создаём
	// 2) Проверяем наличае каждого из хостов в Inventory, если нет - создаём

	organizationOpts := map[string]string{"name": a.organizationName}
	organizations, err := a.client.OrganizationsService.List(ctx, organizationOpts)
	if err != nil {
		return nil, fmt.Errorf("error occured on OrganizationsService.List: %w", err)
	}

	organization, ok := organizations.GetByName(a.organizationName)
	if !ok {
		return nil, fmt.Errorf("organization '%s' not found", a.organizationName)
	}

	inventoryOpts := map[string]string{"name": inventoryName}
	inventories, err := a.client.InventoriesService.ListInventories(ctx, inventoryOpts)
	if err != nil {
		return nil, fmt.Errorf("error occured on list inventories: %w", err)
	}

	inventory, ok := inventories.GetByName(inventoryName)
	if !ok {
		// Если Inventory отсутствует в Playbook, то создадим её.

		inventoryArgs := map[string]interface{}{
			"name":         inventoryName,
			"organization": organization.ID,
		}

		inventory, err = a.client.InventoriesService.CreateInventory(ctx, inventoryArgs)
		if err != nil {
			return nil, fmt.Errorf("error occured on create inventory: %w", err)
		}
	}

	return inventory, nil
}

func (a *Playbook) prepareInventoryGroups(ctx context.Context, inventory *awx.Inventory, ansibleHosts []model.Host) ([]*awx.Group, error) {
	inventoryGroups, err := a.client.GroupService.GetGroupsByInventoryId(ctx, inventory.ID)
	if err != nil {
		return nil, err
	}

	groupsIds := inventoryGroupIds(inventoryGroups)

	groups := groupList(ansibleHosts)

	for _, groupName := range groups {
		if _, ok := groupsIds[groupName]; !ok {
			group, err := a.client.GroupService.CreateGroup(ctx, map[string]interface{}{
				"name":      groupName,
				"inventory": inventory.ID,
			})
			if err != nil {
				return nil, err
			}

			inventoryGroups = append(inventoryGroups, group)
		}
	}

	return inventoryGroups, nil
}

func inventoryGroupIds(inventoryGroups []*awx.Group) map[string]int {
	groupsIds := make(map[string]int)
	for _, group := range inventoryGroups {
		groupsIds[group.Name] = group.ID
	}

	return groupsIds
}

func (a *Playbook) prepareInventoryHosts(ctx context.Context, inventory *awx.Inventory, ansibleHosts []model.Host, inventoryGroups []*awx.Group) error {

	inventoryHosts, err := a.client.HostService.ListInventoryHosts(ctx, inventory.ID)
	if err != nil {
		return err
	}

	groupIds := inventoryGroupIds(inventoryGroups)

	for _, host := range ansibleHosts {
		hostArguments := map[string]interface{}{
			"name":      host.Host,
			"inventory": fmt.Sprint(inventory.ID),
			"variables": JSONUnmarshalString(host.Vars),
		}

		inventoryHost, ok := inventoryHosts.GetByName(host.Host)
		if !ok {
			// Создаём новый хост в Playbook
			inventoryHost, err = a.client.HostService.CreateHost(ctx, hostArguments)
			if err != nil {
				return fmt.Errorf("error occured on inventoryHost create: %w", err)
			}

			if groupId, ok := groupIds[host.Group]; ok {
				err = a.client.GroupService.AddHostToGroup(ctx, groupId, awx.AddHostToGroupBody{
					InventoryID: inventory.ID,
					Name:        host.Host,
				})
			} else {
				logrus.WithFields(map[string]interface{}{
					"Inventory": inventory.Name,
					"Groups":    groupIds,
				}).Warn("Хост [%s] не входит не в одну из групп", host.Host)
			}

		} else {

			// Обновляем информацию о хосте в Playbook
			// ожидаем что если хост уже создан, то он создан в нужной группе
			// TODO: сделать проверку соответсвия всех хостов необходимым группам
			_, err = a.client.HostService.UpdateHost(ctx, inventoryHost.ID, hostArguments)
			if err != nil {
				return fmt.Errorf("error occured on inventoryHost update: %w", err)
			}
		}
	}
	return nil
}

func (a *Playbook) ResetInventory(ctx context.Context, run model.Run) (*awx.Inventory, error) {
	inventory, err := a.prepareInventory(ctx, run.InventoryName)
	if err != nil {
		return nil, err
	}

	groups, err := a.prepareInventoryGroups(ctx, inventory, run.Hosts)
	if err != nil {
		return nil, err
	}

	err = a.prepareInventoryHosts(ctx, inventory, run.Hosts, groups)
	if err != nil {
		return nil, err
	}

	return inventory, nil
}

func (a *Playbook) DeleteInventory(ctx context.Context, name string) error {
	// 1) Сначала удаляем хосты в Inventory
	// 2) Затем удаляем Inventory

	inventoryOpts := map[string]string{"name": name}
	inventories, err := a.client.InventoriesService.ListInventories(ctx, inventoryOpts)
	if err != nil {
		return fmt.Errorf("error occured on list inventories: %w", err)
	}

	inventory, ok := inventories.GetByName(name)
	if !ok {
		// Если Inventory отсутствует в Playbook, то удалять нечего
		return nil
	}

	hosts, err := a.client.HostService.ListInventoryHosts(ctx, inventory.ID)
	if err != nil {
		return err
	}

	for _, host := range hosts.Results {

		// Удаляем host
		err = a.client.HostService.DeleteHost(ctx, host.ID)
		if err != nil {
			return fmt.Errorf("error occured on delete host '%d': %w", host.ID, err)
		}
	}

	groups, err := a.client.GroupService.GetGroupsByInventoryId(ctx, inventory.ID)
	if err != nil {
		return err
	}

	for _, group := range groups {

		// Удаляем группу
		err = a.client.GroupService.DeleteGroup(ctx, group.ID)
		if err != nil {
			return fmt.Errorf("error occured on delete host '%d': %w", group.ID, err)
		}
	}

	// Удаляем Inventory
	err = a.client.InventoriesService.DeleteInventory(ctx, inventory.ID)
	if err != nil {
		return fmt.Errorf("error occured on delete inventory '%d': %w", inventory.ID, err)
	}

	return nil
}

func groupList(hosts []model.Host) []string {
	groupSet := make(map[string]struct{})

	for _, host := range hosts {
		if host.Group != "" {
			groupSet[host.Group] = struct{}{}
		}
	}

	groups := make([]string, 0, len(groupSet))
	for group := range groupSet {
		groups = append(groups, group)
	}

	return groups
}
