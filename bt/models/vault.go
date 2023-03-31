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

type VaultUsernamePasswordAccount struct {
	ID             types.String `tfsdk:"id"`
	Type           types.String `tfsdk:"type"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Personal       types.Bool   `tfsdk:"personal"`
	OwnerUserID    types.Int64  `tfsdk:"owner_user_id"`
	AccountGroupID types.Int64  `tfsdk:"account_group_id"`
	AccountPolicy  types.String `tfsdk:"account_policy"`

	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password" sra:"persist_state"`
	LastCheckoutTimestamp types.String `tfsdk:"last_checkout_timestamp"`
}

type VaultSSHAccount struct {
	ID             types.String `tfsdk:"id"`
	Type           types.String `tfsdk:"type"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Personal       types.Bool   `tfsdk:"personal"`
	OwnerUserID    types.Int64  `tfsdk:"owner_user_id"`
	AccountGroupID types.Int64  `tfsdk:"account_group_id"`
	AccountPolicy  types.String `tfsdk:"account_policy"`

	Username              types.String `tfsdk:"username"`
	PublicKey             types.String `tfsdk:"public_key"`
	PrivateKey            types.String `tfsdk:"private_key" sra:"persist_state"`
	PrivateKeyPassphrase  types.String `tfsdk:"private_key_passphrase" sra:"persist_state"`
	PrivateKeyPublicCert  types.String `tfsdk:"private_key_public_cert"`
	LastCheckoutTimestamp types.String `tfsdk:"last_checkout_timestamp"`
}

type VaultAccountGroup struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	AccountPolicy types.String `tfsdk:"account_policy"`
}
