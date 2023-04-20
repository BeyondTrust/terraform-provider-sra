# Manage example Jump Client Installer

resource "sra_jump_client_installer" "example" {
  jump_group_id = data.sra_jump_group_list.jg.items[0].id
}

data "sra_jump_group_list" "jg" {
  code_name = "example_group"
}

// Output installer information

output "client" {
  value = {
    "installer_id": sra_jump_client_installer.test_jc.installer_id,
    "key_info": sra_jump_client_installer.test_jc.key_info,
    "expires": sra_jump_client_installer.test_jc.expiration_timestamp
  }
}