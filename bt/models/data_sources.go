package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type JumpGroupModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	CodeName   types.String `tfsdk:"code_name"`
	Comments   types.String `tfsdk:"comments"`
	EcmGroupId types.Int64  `tfsdk:"ecm_group_id"`
}
