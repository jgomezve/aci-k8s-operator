package aci

import (
	"fmt"
	"strconv"

	"github.com/ciscoecosystem/aci-go-client/client"
	"github.com/ciscoecosystem/aci-go-client/models"
)

type ApicClient struct {
	host     string
	user     string
	password string
	client   *client.Client
}

type ApicInterface interface {
	CreateTenant(name, description string) error
	DeleteTenant(name string) error
	CreateApplicationProfile(name, description, tenantName string) error
	DeleteApplicationProfile(name, tenantName string) error
	CreateEndpointGroup(name, description, appName, tenantName string) error
	DeleteEndpointGroup(name, appName, tenantName string) error
	CreateFilterAndFilterEntry(tenantName, name, eth, ip string, port int) error
	DeleteFilter(name, tenantName string) error
	AddTagAnnotation(key, value, parentDn string) error
	FilterExists(name, tenantName string) (bool, error)
	CreateContract(tenantName, name string, filters []string) error
	DeleteContract(tenantName, name string) error
	EpgExists(name, appName, tenantName string) (bool, error)
	AddTagAnnotationToEpg(name, appName, tenantName, key, value string) error
	GetEpgWithAnnotation(appName, tenantName, key string) ([]string, error)
}

func NewApicClient(host, user, password string) (*ApicClient, error) {
	ac := &ApicClient{
		host:     host,
		user:     user,
		password: password,
		client:   client.GetClient(fmt.Sprintf("https://%s/", host), user, client.Password(password), client.Insecure(true)),
	}
	// TODO: Re-use connection
	return ac, nil
}

func (ac *ApicClient) CreateTenant(name, description string) error {
	fvTenantAttr := models.TenantAttributes{}
	fvTenant := models.NewTenant(fmt.Sprintf("tn-%s", name), "uni", description, fvTenantAttr)
	err := ac.client.Save(fvTenant)
	if err != nil {
		return err
	}
	return nil
}

func (ac *ApicClient) DeleteTenant(name string) error {
	err := ac.client.DeleteTenant(name)
	if err != nil {
		return err
	}
	return nil
}

func (ac *ApicClient) CreateApplicationProfile(name, description, tenantName string) error {
	fvAppAttr := models.ApplicationProfileAttributes{}
	fvAppAttr.Annotation = "orchestrator:kubernetes"
	fvApp := models.NewApplicationProfile(fmt.Sprintf("ap-%s", name), fmt.Sprintf("uni/tn-%s", tenantName), description, fvAppAttr)
	err := ac.client.Save(fvApp)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	return nil
}

func (ac *ApicClient) DeleteApplicationProfile(name, tenantName string) error {
	err := ac.client.DeleteApplicationProfile(name, tenantName)
	if err != nil {
		return err
	}
	return nil
}

func (ac *ApicClient) CreateEndpointGroup(name, description, appName, tenantName string) error {

	fvAEpgAttr := models.ApplicationEPGAttributes{}
	fvAEpgAttr.Annotation = "orchestrator:kubernetes"
	fvAEpg := models.NewApplicationEPG(fmt.Sprintf("epg-%s", name), fmt.Sprintf("uni/tn-%s/ap-%s", tenantName, appName), description, fvAEpgAttr)

	err := ac.client.Save(fvAEpg)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}
	return nil
}

func (ac *ApicClient) DeleteEndpointGroup(name, appName, tenantName string) error {
	err := ac.client.DeleteApplicationEPG(name, appName, tenantName)
	if err != nil {
		return err
	}
	return nil
}

func (ac *ApicClient) EpgExists(name, appName, tenantName string) (bool, error) {

	fvAEPgCont, err := ac.client.Get(fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, name))
	if err != nil {
		return false, err
	}
	fvAEPg := models.ApplicationEPGFromContainer(fvAEPgCont)

	if fvAEPg.DistinguishedName == "" {
		return false, nil
	}

	return true, nil
}

func (ac *ApicClient) AddTagAnnotationToEpg(name, appName, tenantName, key, value string) error {

	parentDn := fmt.Sprintf("uni/tn-%s/ap-%s/epg-%s", tenantName, appName, name)
	if err := ac.AddTagAnnotation(key, value, parentDn); err != nil {
		return err
	}
	return nil
}

func (ac *ApicClient) GetEpgWithAnnotation(appName, tenantName, key string) ([]string, error) {

	epgs := []string{}
	epgList, err := ac.client.ListApplicationEPG(appName, tenantName)
	if err != nil {
		return []string{}, err
	}

	for _, epg := range epgList {
		_, err := ac.client.ReadAnnotation(key, epg.DistinguishedName)
		if err == nil {
			epgs = append(epgs, epg.Name)
		}
	}

	return epgs, nil
}
func (ac *ApicClient) CreateContract(tenantName, name string, filters []string) error {
	vzBrCPAttr := models.ContractAttributes{}
	vzBrCPAttr.Name = name
	vzBrCPAttr.Annotation = "orchestrator:kubernetes"

	vzSubjAttr := models.ContractSubjectAttributes{}
	vzSubjAttr.Name = name
	vzSubjAttr.RevFltPorts = "yes"

	vzBrCP := models.NewContract(fmt.Sprintf("brc-%s", name), fmt.Sprintf("uni/tn-%s", tenantName), "", vzBrCPAttr)
	err := ac.client.Save(vzBrCP)
	if err != nil {
		return err
	}
	vzSubj := models.NewContractSubject(fmt.Sprintf("subj-%s", name), vzBrCP.DistinguishedName, "", vzSubjAttr)
	err = ac.client.Save(vzSubj)
	if err != nil {
		return err
	}
	for _, flt := range filters {
		err = ac.client.CreateRelationvzRsSubjFiltAttFromContractSubject(vzSubj.DistinguishedName, flt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ac *ApicClient) DeleteContract(tenantName, name string) error {
	err := ac.client.DeleteContract(name, tenantName)
	if err != nil {
		return err
	}
	return nil
}

func (ac *ApicClient) CreateFilterAndFilterEntry(tenantName, name, eth, ip string, port int) error {

	vzFilterAttr := models.FilterAttributes{}
	vzFilterAttr.Annotation = "orchestrator:kubernetes"

	vzEntryAttr := models.FilterEntryAttributes{}
	vzEntryAttr.EtherT = eth
	vzEntryAttr.Prot = ip
	vzEntryAttr.DFromPort = strconv.Itoa(port)
	vzEntryAttr.DToPort = strconv.Itoa(port)

	fvFilter := models.NewFilter(fmt.Sprintf("flt-%s", name), fmt.Sprintf("uni/tn-%s", tenantName), "", vzFilterAttr)
	err := ac.client.Save(fvFilter)
	if err != nil {
		return err
	}
	fvFilterEntry := models.NewFilterEntry(fmt.Sprintf("e-%s", name), fvFilter.DistinguishedName, "", vzEntryAttr)
	err = ac.client.Save(fvFilterEntry)
	if err != nil {
		return err
	}
	return nil
}

func (ac *ApicClient) DeleteFilter(tenantName, name string) error {
	err := ac.client.DeleteFilter(name, tenantName)
	if err != nil {
		return err
	}
	return nil
}

func (ac *ApicClient) AddTagAnnotation(key, value, parentDn string) error {
	tag := models.NewAnnotation(fmt.Sprintf("annotationKey-[%s]", key), parentDn, models.AnnotationAttributes{Key: key, Value: value})
	err := ac.client.Save(tag)
	if err != nil {
		return err
	}
	return nil
}

func (ac *ApicClient) FilterExists(name, tenantName string) (bool, error) {
	return true, nil
}
