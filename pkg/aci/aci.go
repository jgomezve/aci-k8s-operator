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
