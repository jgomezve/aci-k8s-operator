package aci

import (
	"fmt"

	"github.com/ciscoecosystem/aci-go-client/client"
	"github.com/ciscoecosystem/aci-go-client/models"
)

type ApicClient struct {
	host     string
	user     string
	password string
	client   *client.Client
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
	fvApp := models.NewApplicationProfile(fmt.Sprintf("ap-%s", name), fmt.Sprintf("uni/tn-%s", tenantName), description, fvAppAttr)
	err := ac.client.Save(fvApp)
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
	fvAEpg := models.ApplicationEPGAttributes{}
	fvAEpg.Annotation = "orchestrator:kubernetes"
	fvApp := models.NewApplicationEPG(fmt.Sprintf("epg-%s", name), fmt.Sprintf("uni/tn-%s/ap-%s", tenantName, appName), description, fvAEpg)
	err := ac.client.Save(fvApp)
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
