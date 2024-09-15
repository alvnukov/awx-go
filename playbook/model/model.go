package model

type Host struct {
	Host  string // FQDN
	Group string
	Vars  map[string]string
}

type Run struct {
	InventoryName string
	TemplateName  string
	Timeout       int
	Hosts         []Host
	Vars          map[string]interface{}
}

func (r Run) HostList() []string {
	var hosts []string
	for _, host := range r.Hosts {
		if host.Host == "" {
			continue
		}
		hosts = append(hosts, host.Host)
	}
	return hosts
}
