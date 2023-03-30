package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VaultAccount struct {
	ID             types.String `tfsdk:"id"`
	Type           types.String `tfsdk:"type"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Personal       types.Bool   `tfsdk:"personal"`
	OwnerUserID    types.Int64  `tfsdk:"owner_user_id"`
	AccountGroupID types.Int64  `tfsdk:"account_group_id"`
	AccountPolicy  types.String `tfsdk:"account_policy"`
}
