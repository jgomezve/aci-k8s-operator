package aci

import (
	"fmt"
)

type ApicClientMocks struct {
	filters []string
}

func init() {
	ApicMockClient.filters = []string{}
}

var (
	ApicMockClient ApicClientMocks
)

func (ac *ApicClientMocks) CreateTenant(name, description string) error {
	fmt.Print("Creating Tenant\n")
	return nil
}

func (ac *ApicClientMocks) DeleteTenant(name string) error {
	fmt.Print("Deleting Tenant\n")
	return nil
}

func (ac *ApicClientMocks) CreateApplicationProfile(name, description, tenantName string) error {
	return nil
}

func (ac *ApicClientMocks) DeleteApplicationProfile(name, tenantName string) error {
	return nil
}

func (ac *ApicClientMocks) CreateEndpointGroup(name, description, appName, tenantName string) error {
	return nil
}

func (ac *ApicClientMocks) DeleteEndpointGroup(name, appName, tenantName string) error {
	return nil
}

func (ac *ApicClientMocks) EpgExists(name, appName, tenantName string) (bool, error) {
	return true, nil
}

func (ac *ApicClientMocks) AddTagAnnotationToEpg(name, appName, tenantName, key, value string) error {
	return nil
}
func (ac *ApicClientMocks) CreateFilterAndFilterEntry(tenantName, name, eth, ip string, port int) error {
	if !ac.FilterExists(name) {
		fmt.Printf("Creating Filter %s in Tenant %s\n", name, tenantName)
		ac.filters = append(ac.filters, name)
	}
	return nil
}

func (ac *ApicClientMocks) GetEpgWithAnnotation(appName, tenantName, key string) ([]string, error) {
	return []string{}, nil
}

func (ac *ApicClientMocks) DeleteFilter(tenantName, name string) error {
	fmt.Printf("Deleting Filter %s in Tenant %s\n", name, tenantName)
	for i, v := range ac.filters {
		if v == name {
			ac.filters = append(ac.filters[:i], ac.filters[i+1:]...)
			break
		}
	}
	return nil
}

func (ac *ApicClientMocks) FilterExists(name string) bool {
	for _, flt := range ac.filters {
		if flt == name {
			return true
		}
	}
	return false
}

func (ac *ApicClientMocks) CreateContract(tenantName, name string, filters []string) error {
	return nil
}
func (ac *ApicClientMocks) DeleteContract(tenantName, name string) error {
	return nil
}

func (ac *ApicClientMocks) AddTagAnnotation(key, value, parentDn string) error {
	return nil
}
