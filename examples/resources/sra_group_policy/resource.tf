resource "sra_group_policy" "example" {
  name = "Terraform Managed Group Policy"

  perm_jump_client          = true
  perm_remote_rdp           = true
  perm_session_idle_timeout = 900
}
