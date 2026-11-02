package main

import (
	"context"
	"log"
	"terraform-provider-sra/bt"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Provider documentation generation. --tf-version pins the Terraform that
// tfplugindocs downloads to read the schema: one older than 1.11 drops the
// write-only marking from password_wo.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name sra --tf-version 1.13.4
//go:generate go run openapi.go

func main() {
	err := providerserver.Serve(context.Background(), bt.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/beyondtrust/sra",
	})

	if err != nil {
		log.Fatal(err)
	}
}
