package aci

import (
	"fmt"

	"github.com/jgomezve/aci-k8s-operator/pkg/utils"
)

type endpointGroup struct {
	name      string
	tnt       string
	app       string
	Bd        string
	Vmm       string
	tags      map[string]string
	contracts map[string][]string
	// Only one master
	Master []string
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

func (ac *ApicClientMocks) EmptyApplicationProfile(name, tenantName string) (bool, error) {
	count := 0
	for _, epg := range ac.endpointGroups {
		if epg.app == name && epg.tnt == tenantName {
			count = count + 1
		}
	}
	return count == 0, nil
}

func (ac *ApicClientMocks) CreateEndpointGroup(name, description, appName, tenantName, bdName, vmmName string) error {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, name)
	fmt.Printf("Creating EPG %s \n", dn)
	ac.endpointGroups[dn] = endpointGroup{name: name, app: appName, tnt: tenantName, Bd: bdName, Vmm: vmmName, tags: map[string]string{}, contracts: map[string][]string{}, Master: []string{}}
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

// Contract in the same tenant
func (ac *ApicClientMocks) ConsumeContract(epgName, appName, tenantName, conName string) error {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, epgName)
	fmt.Printf("EPG %s consuming contract %s\n", dn, conName)
	if !utils.Contains(ac.endpointGroups[dn].contracts["consumed"], conName) {
		ac.endpointGroups[dn].contracts["consumed"] = append(ac.endpointGroups[dn].contracts["consumed"], conName)
	}
	return nil
}

// Contract in the same tenant
func (ac *ApicClientMocks) ProvideContract(epgName, appName, tenantName, conName string) error {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, epgName)
	fmt.Printf("EPG %s providing contract %s\n", dn, conName)
	if !utils.Contains(ac.endpointGroups[dn].contracts["provided"], conName) {
		ac.endpointGroups[dn].contracts["provided"] = append(ac.endpointGroups[dn].contracts["provided"], conName)
	}
	return nil
}

func (ac *ApicClientMocks) GetContracts(epgName, appName, tenantName string) (map[string][]string, error) {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, epgName)
	fmt.Printf("Contracts consumed/provided by EPG %s : %s\n", dn, ac.endpointGroups[dn].contracts)
	return ac.endpointGroups[dn].contracts, nil
}

func (ac *ApicClientMocks) DeleteContractConsumer(epgName, appName, tenantName, conName string) error {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, epgName)
	fmt.Printf("EPG %s no longer consuming contract %s\n", dn, conName)
	ac.endpointGroups[dn].contracts["consumed"] = utils.Remove(ac.endpointGroups[dn].contracts["consumed"], conName)
	return nil
}

func (ac *ApicClientMocks) DeleteContractProvider(epgName, appName, tenantName, conName string) error {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, epgName)
	fmt.Printf("EPG %s no longer providing contract %s\n", dn, conName)
	ac.endpointGroups[dn].contracts["provided"] = utils.Remove(ac.endpointGroups[dn].contracts["provided"], conName)
	return nil
}

func (ac *ApicClientMocks) InheritContractFromMaster(epgName, appName, tenantName, appMasterName, epgMasterName string) error {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, epgName)
	fmt.Printf("EPG %s inheriting contract from master %s/%s \n", dn, appMasterName, epgMasterName)
	currentEpgConf := ac.endpointGroups[dn]
	currentEpgConf.Master = append(currentEpgConf.Master, fmt.Sprintf("%s/%s", appMasterName, epgMasterName))
	ac.endpointGroups[dn] = currentEpgConf
	return nil
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

// Function only available in the Mock
func (ac *ApicClientMocks) GetEpg(name, appName, tenantName string) endpointGroup {
	dn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, name)
	fmt.Printf("Getting EPG %s \n", dn)
	return ac.endpointGroups[dn]
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
	_, exists := ac.contracts[dn]
	// If the contracts exists, then append new filters
	if !exists {
		ac.contracts[dn] = contract{name: name, tnt: tenantName, filters: filters}
	} else {
		fmt.Printf("Contract %s already exists\n", dn)
		for _, flt := range filters {
			if !utils.Contains(ac.contracts[dn].filters, flt) {
				currentFilters := ac.contracts[dn].filters
				currentFilters = append(currentFilters, flt)
				fmt.Printf("Adding filter %s to contract %s\n", flt, dn)
				ac.contracts[dn] = contract{name: name, tnt: tenantName, filters: currentFilters}
			}
		}
	}
	return nil
}

func (ac *ApicClientMocks) GetContractFilters(contractName, tenantName string) ([]string, error) {
	dn := fmt.Sprintf("uni/tn-%s/brp-%s", tenantName, contractName)
	return ac.contracts[dn].filters, nil
}

func (ac *ApicClientMocks) DeleteFilterFromSubjectContract(subjectName, tenantName, filter string) error {
	dn := fmt.Sprintf("uni/tn-%s/brp-%s", tenantName, subjectName)
	fmt.Printf("Deleting filter %s from contract %s\n", filter, subjectName)
	ftls := ac.contracts[dn].filters
	ac.contracts[dn] = contract{name: subjectName, tnt: tenantName, filters: utils.Remove(ftls, filter)}
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
