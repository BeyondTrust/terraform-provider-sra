resource "sra_group_policy_member" "supplier_saml_group" {
  group_policy_id      = "9"
  security_provider_id = 2
  group_name           = "Suppliers"
}
