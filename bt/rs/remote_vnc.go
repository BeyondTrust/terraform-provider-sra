package rs

import (
	"context"
	"terraform-provider-beyondtrust-sra/api"
	"terraform-provider-beyondtrust-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// These throw away variable declarations are to allow the compiler to
// enforce compliance to these interfaces
var (
	_ resource.Resource                = &remoteVNCResource{}
	_ resource.ResourceWithConfigure   = &remoteVNCResource{}
	_ resource.ResourceWithImportState = &remoteVNCResource{}
	// _ resource.ResourceWithModifyPlan  = &remoteVNCResource{}
)

// Factory function to generate a new resource type. This must be in the
// main list of resource functions in api_resource.go
func newRemoteVNCResource() resource.Resource {
	return &remoteVNCResource{}
}

// The main type for the resource. By convention this should be in the form
//
//	<resourceName>Resource
//
// This type name of the API model will be used to generate the public name
// for this resource that is used in the *.tf files. The
// public name will be converted like:
//
//	ResourceName -> sra_resource_name
type remoteVNCResource struct {
	// Compose with the main apiResource struct to get all the boilerplate
	// implementations. Types are: this resource, api model, terraform model
	apiResource[api.RemoteVNC, models.RemoteVNCModel]
}

// We must define the schema for each resource individually. Anything that can be supplied by the API response
// needs to be marked as "Computed", even if we translate from "null" on a POST to an empty string
func (r *remoteVNCResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remote VNC Jump Item.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"jumpoint_id": schema.Int64Attribute{
				Required: true,
			},
			"hostname": schema.StringAttribute{
				Required: true,
			},
			"port": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(5900),
			},
			"jump_group_id": schema.Int64Attribute{
				Required: true,
			},
			"jump_group_type": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString("shared"),
				Validators: jumpGroupTypeValidator(),
			},
			"tag": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"comments": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"jump_policy_id": schema.Int64Attribute{
				Optional: true,
			},
			"session_policy_id": schema.Int64Attribute{
				Optional: true,
			},
		},
	}
}
