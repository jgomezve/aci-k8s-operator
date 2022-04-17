package aci

import (
	"fmt"
)

type ApicClientMocks struct {
	Ip string
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

func (ac *ApicClientMocks) CreateFilterAndFilterEntry(tenantName, name, eth, ip string, port int) error {
	fmt.Printf("Creating Filter %s in Tenant %s\n", name, tenantName)
	return nil
}

func (ac *ApicClientMocks) DeleteFilter(tenantName, name string) error {
	fmt.Printf("Deleting Filter %s in Tenant %s\n", name, tenantName)
	return nil
}

func (ac *ApicClientMocks) AddTagAnnotation(key, value, parentDn string) error {
	return nil
}
