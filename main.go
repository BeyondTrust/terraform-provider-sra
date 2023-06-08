package main

import (
	"context"
	"log"
	"terraform-provider-sra/bt"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Provider documentation generation.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name sra

func main() {
	err := providerserver.Serve(context.Background(), bt.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/beyondtrust/sra",
	})

	if err != nil {
		log.Fatal(err)
	}
}
