package test

import (
	"terraform-provider-beyondtrust-sra/bt"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	providerConfig = `
provider "sra" {
	host = "mpam.dev.bomgar.com"
	client_id = "114635791a8bc6e21d813d5385d100afcb883a2d"
	client_secret = "wUwZTVwC0Erh3/01TcG41TbWHcntMgdRZHkhqcwNKYQK"
}
`
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"sra": providerserver.NewProtocol6WithError(bt.New()),
	}
)
