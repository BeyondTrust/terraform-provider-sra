# List all Jump Client Installers
data "sra_jump_client_installer_list" "all" {}

# Filter by tag
data "sra_jump_client_installer_list" "filtered" {
  tag = "example"
}

