package aci

import (
	"fmt"
)

type endpointGroup struct {
	name string
	tnt  string
	app  string
}

type contract struct {
	name    string
	tnt     string
	filters []string
}

type filter struct {
	name string
	tnt  string
}

// func (epg endpointGroup) getDn() string {
// 	return fmt.Sprintf("uni\tn-%s/ap-%s/epg-%s", epg.tnt, epg.app, epg.name)
// }

type ApicClientMocks struct {
	filters        map[string]filter
	endpointGroups map[string]endpointGroup
	contracts      map[string]contract
}

func init() {
	ApicMockClient.filters = map[string]filter{}
	ApicMockClient.endpointGroups = map[string]endpointGroup{}
	ApicMockClient.contracts = map[string]contract{}
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
	fmt.Printf("Creating Application Profile %s in Tenant %s\n", name, tenantName)
	return nil
}

func (ac *ApicClientMocks) DeleteApplicationProfile(name, tenantName string) error {
	return nil
}

func (ac *ApicClientMocks) CreateEndpointGroup(name, description, appName, tenantName string) error {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, name)
	fmt.Printf("Creating EPG %s \n", dn)
	ac.endpointGroups[dn] = endpointGroup{name: name, app: appName, tnt: tenantName}
	return nil
}

func (ac *ApicClientMocks) DeleteEndpointGroup(name, appName, tenantName string) error {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, name)
	delete(ac.endpointGroups, dn)
	fmt.Printf("Deleting EPG %s \n", dn)
	return nil
}

func (ac *ApicClientMocks) EpgExists(name, appName, tenantName string) (bool, error) {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, name)
	fmt.Printf("Checking if EPG %s exists\n", dn)
	_, exists := ac.endpointGroups[dn]
	return exists, nil
}

func (ac *ApicClientMocks) AddTagAnnotationToEpg(name, appName, tenantName, key, value string) error {
	return nil
}
func (ac *ApicClientMocks) CreateFilterAndFilterEntry(tenantName, name, eth, ip string, port int) error {
	dn := fmt.Sprintf("uni/tn-%s/flt-%s", tenantName, name)
	fmt.Printf("Creating Filter %s \n", dn)
	ac.filters[dn] = filter{name: name, tnt: tenantName}
	return nil
}

func (ac *ApicClientMocks) GetEpgWithAnnotation(appName, tenantName, key string) ([]string, error) {
	return []string{}, nil
}

func (ac *ApicClientMocks) DeleteFilter(tenantName, name string) error {
	dn := fmt.Sprintf("uni/tn-%s/flt-%s", tenantName, name)
	fmt.Printf("Deleting Filter %s \n", dn)
	delete(ac.filters, dn)
	return nil
}

func (ac *ApicClientMocks) FilterExists(name, tenantName string) (bool, error) {
	dn := fmt.Sprintf("uni/tn-%s/flt-%s", tenantName, name)
	fmt.Printf("Checking if Filter %s exists\n", dn)
	_, exists := ac.filters[dn]
	return exists, nil
}

func (ac *ApicClientMocks) CreateContract(tenantName, name string, filters []string) error {
	dn := fmt.Sprintf("uni/tn-%s/brp-%s", tenantName, name)
	fmt.Printf("Creating contract %s\n", dn)
	ac.contracts[dn] = contract{name: name, tnt: tenantName, filters: filters}
	return nil
}
func (ac *ApicClientMocks) DeleteContract(tenantName, name string) error {
	return nil
}

func (ac *ApicClientMocks) AddTagAnnotation(key, value, parentDn string) error {
	return nil
}
