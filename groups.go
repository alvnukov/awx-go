package awx

import (
	"context"
	"fmt"
)

// HostService implements awx Hosts apis.
type GroupService struct {
	Requester *Requester
}

type AddHostToGroupBody struct {
	InventoryID int    `json:"inventory"`
	Name        string `json:"name"`
}

func (g *GroupService) AddHostToGroup(ctx context.Context, groupId int, body AddHostToGroupBody) error {
	endpoint := fmt.Sprintf("/api/v2/groups/%d/hosts/", groupId)

	_, err := g.Requester.Post(ctx, endpoint, body, nil)
	if err != nil {
		return err
	}

	return nil
}

// ListGroups возвращает группы которые созданы у inventoryId
func (g *GroupService) GetGroupsByInventoryId(ctx context.Context, inventoryId int) ([]*Group, error) {
	result := ListGroups{}
	endpoint := fmt.Sprintf("/api/v2/inventories/%d/groups/", inventoryId)

	_, err := g.Requester.Get(ctx, endpoint, &result, nil)
	if err != nil {
		return nil, err
	}

	// if err := CheckResponse(resp); err != nil {
	// 	return nil, result, err
	// }

	return result.Results, nil
}

// ListGroups shows list of awx Groups.
func (g *GroupService) ListGroups(ctx context.Context, params map[string]string) (*ListGroups, error) {
	result := ListGroups{}
	endpoint := "/api/v2/groups/"

	_, err := g.Requester.Get(ctx, endpoint, &result, params)
	if err != nil {
		return nil, err
	}

	// if err := CheckResponse(resp); err != nil {
	// 	return nil, result, err
	// }

	return &result, nil
}

// CreateGroup creates an awx Group.
func (g *GroupService) CreateGroup(ctx context.Context, data map[string]interface{}) (*Group, error) {
	result := Group{}
	endpoint := "/api/v2/groups/"

	validate, status := ValidateParams(data, []string{"name", "inventory"})
	if !status {
		return nil, fmt.Errorf("mandatory input arguments are absent: %s", validate)
	}

	_, err := g.Requester.Post(ctx, endpoint, data, &result)
	if err != nil {
		return nil, err
	}

	// if err := CheckResponse(resp); err != nil {
	// 	return nil, err
	// }

	return &result, nil
}

// UpdateGroup update an awx group
func (g *GroupService) UpdateGroup(ctx context.Context, id int, data map[string]interface{}) (*Group, error) {
	result := new(Group)
	endpoint := fmt.Sprintf("/api/v2/groups/%d", id)

	_, err := g.Requester.Patch(ctx, endpoint, data, &result)
	if err != nil {
		return nil, err
	}

	// if err := CheckResponse(resp); err != nil {
	// 	return nil, err
	// }

	return result, nil
}

// DeleteGroup delete an awx Group.
func (g *GroupService) DeleteGroup(ctx context.Context, id int) error {
	endpoint := fmt.Sprintf("/api/v2/groups/%d", id)

	_, err := g.Requester.Delete(ctx, endpoint)
	if err != nil {
		return err
	}

	// if err := CheckResponse(resp); err != nil {
	// 	return nil, err
	// }

	return nil
}

func (g *GroupService) ListGroupHosts(ctx context.Context, group *Group) ([]*Host, error) {
	result := ListHosts{}
	endpoint := group.Related.Hosts

	_, err := g.Requester.Get(ctx, endpoint, result, make(map[string]string))
	if err != nil {
		return nil, err
	}

	return result.Results, nil
}
