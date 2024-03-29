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

	JumpItemAssociation    types.Object `tfsdk:"jump_item_association"`
	GroupPolicyMemberships types.Set    `tfsdk:"group_policy_memberships"`
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

	JumpItemAssociation    types.Object `tfsdk:"jump_item_association"`
	GroupPolicyMemberships types.Set    `tfsdk:"group_policy_memberships"`
}

type VaultSSHAccountDS struct {
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
	PrivateKeyPublicCert  types.String `tfsdk:"private_key_public_cert"`
	LastCheckoutTimestamp types.String `tfsdk:"last_checkout_timestamp"`
}

type VaultTokenAccount struct {
	ID             types.String `tfsdk:"id"`
	Type           types.String `tfsdk:"type"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Personal       types.Bool   `tfsdk:"personal"`
	OwnerUserID    types.Int64  `tfsdk:"owner_user_id"`
	AccountGroupID types.Int64  `tfsdk:"account_group_id"`
	AccountPolicy  types.String `tfsdk:"account_policy"`

	Token                 types.String `tfsdk:"token" sra:"persist_state"`
	LastCheckoutTimestamp types.String `tfsdk:"last_checkout_timestamp"`

	JumpItemAssociation    types.Object `tfsdk:"jump_item_association"`
	GroupPolicyMemberships types.Set    `tfsdk:"group_policy_memberships"`
}

type VaultAccountGroup struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	AccountPolicy types.String `tfsdk:"account_policy"`

	JumpItemAssociation    types.Object `tfsdk:"jump_item_association"`
	GroupPolicyMemberships types.Set    `tfsdk:"group_policy_memberships"`
}

type VaultAccountGroupDS struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	AccountPolicy types.String `tfsdk:"account_policy"`
}

type VaultAccountPolicy struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	CodeName                  types.String `tfsdk:"code_name"`
	Description               types.String `tfsdk:"description"`
	AutoRotateCredentials     types.Bool   `tfsdk:"auto_rotate_credentials"`
	AllowSimultaneousCheckout types.Bool   `tfsdk:"allow_simultaneous_checkout"`
	ScheduledPasswordRotation types.Bool   `tfsdk:"scheduled_password_rotation"`
	MaximumPasswordAge        types.Int64  `tfsdk:"maximum_password_age"`
}

type VaultSecret struct {
	ID               types.String `tfsdk:"id"`
	Username         types.String `tfsdk:"username"`
	Type             types.String `tfsdk:"type"`
	Secret           types.String `tfsdk:"secret"`
	SignedPublicCert types.String `tfsdk:"signed_public_cert"`
}
