resource "sra_group_policy" "example" {
  name = "Terraform Managed Group Policy"

  perm_collaborate          = true
  perm_session_idle_timeout = 900
}
