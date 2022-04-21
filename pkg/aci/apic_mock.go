package aci

import (
	"fmt"
)

type endpointGroup struct {
	name string
	tnt  string
	app  string
	tags map[string]string
}

type contract struct {
	name    string
	tnt     string
	filters []string
}

type filter struct {
	name string
	tnt  string
	tags map[string]string
}

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
	ac.endpointGroups[dn] = endpointGroup{name: name, app: appName, tnt: tenantName, tags: map[string]string{}}
	return nil
}

func (ac *ApicClientMocks) DeleteEndpointGroup(name, appName, tenantName string) error {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, name)
	fmt.Printf("Deleting EPG %s \n", dn)
	delete(ac.endpointGroups, dn)
	return nil
}

func (ac *ApicClientMocks) EpgExists(name, appName, tenantName string) (bool, error) {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, name)
	fmt.Printf("Checking if EPG %s exists\n", dn)
	_, exists := ac.endpointGroups[dn]
	return exists, nil
}

func (ac *ApicClientMocks) AddTagAnnotationToEpg(name, appName, tenantName, key, value string) error {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, name)
	fmt.Printf("Add Annotation {%s:%s} to EPG %s\n", key, value, dn)
	ac.endpointGroups[dn].tags[key] = value
	return nil
}

func (ac *ApicClientMocks) RemoveTagAnnotation(name, appName, tenantName, key string) error {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, name)
	fmt.Printf("Remove Annotation {%s: x } to EPG %s \n", key, dn)
	delete(ac.endpointGroups[dn].tags, key)
	return nil
}

func (ac *ApicClientMocks) CreateFilterAndFilterEntry(tenantName, name, eth, ip string, port int) error {
	dn := fmt.Sprintf("uni/tn-%s/flt-%s", tenantName, name)
	fmt.Printf("Creating Filter %s \n", dn)
	ac.filters[dn] = filter{name: name, tnt: tenantName, tags: map[string]string{}}
	return nil
}

func (ac *ApicClientMocks) GetEpgWithAnnotation(appName, tenantName, key string) ([]string, error) {
	fmt.Printf("Getting EPG with tag %s \n", key)
	epgList := []string{}
	for _, epg := range ac.endpointGroups {
		for k, _ := range epg.tags {
			if k == key {
				epgList = append(epgList, epg.name)
			}
		}
	}
	return epgList, nil
}

func (ac *ApicClientMocks) GetAnnotationsEpg(name, appName, tenantName string) ([]string, error) {
	keys := []string{}
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, name)
	fmt.Printf("Getting tags of EPG %s \n", dn)
	for k, _ := range ac.endpointGroups[dn].tags {
		keys = append(keys, k)
	}
	return keys, nil
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
	dn := fmt.Sprintf("uni/tn-%s/brp-%s", tenantName, name)
	fmt.Printf("Deleting Contract %s \n", dn)
	delete(ac.contracts, dn)
	return nil
}

func (ac *ApicClientMocks) AddTagAnnotationToFilter(name, tenantName, key, value string) error {
	dn := fmt.Sprintf("uni/tn-%s/flt-%s", tenantName, name)
	fmt.Printf("Add Annotation {%s:%s} to Filter %s\n", key, value, dn)
	ac.filters[dn].tags[key] = value
	return nil
}

func (ac *ApicClientMocks) GetFilterWithAnnotation(tenantName, key string) ([]string, error) {
	fmt.Printf("Getting Filters with tag %s \n", key)
	filterList := []string{}
	for _, flt := range ac.filters {
		for k, _ := range flt.tags {
			if k == key {
				filterList = append(filterList, flt.name)
			}
		}
	}
	return filterList, nil
}
