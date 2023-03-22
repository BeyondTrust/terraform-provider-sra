package main

import (
	"context"
	"terraform-provider-beyondtrust-sra/bt"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	providerserver.Serve(context.Background(), bt.New, providerserver.ServeOpts{
		Address: "hashicorp.com/edu/beyondtrust-sra",
	})
}
