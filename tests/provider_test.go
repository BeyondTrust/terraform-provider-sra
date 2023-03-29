package test

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"terraform-provider-beyondtrust-sra/bt"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func readTFVars() string {
	f, err := os.Open("../test-tf-files/terraform.tfvars")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()
	lines := []string{"locals {"}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	lines = append(lines, "}")

	return strings.Join(lines, "\n")
}

var (
	providerConfig = fmt.Sprintf(`
%s
provider "sra" {
	host = local.api_auth.host
	client_id = local.api_auth.client_id
	client_secret = local.api_auth.client_secret
}
`, readTFVars())
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"sra": providerserver.NewProtocol6WithError(bt.New()),
	}
)
