terraform {
  required_providers {
    sra = {
      source = "beyondtrust/beyondtrust-sra"
    }
  }
}

// Configuration

data "sra_shell_jump_list" "sj" {}
data "sra_protocol_tunnel_jump_list" "pt" {}
data "sra_remote_rdp_list" "rdp" {}
data "sra_remote_vnc_list" "vnc" {}
data "sra_web_jump_list" "web" {}
output "all" {
  value = {
    PrtocolTunnelJump = data.sra_protocol_tunnel_jump_list.pt.items
    ShellJump         = data.sra_shell_jump_list.sj.items
    RemoteRDP         = data.sra_remote_rdp_list.rdp.items
    RemoteVNC         = data.sra_remote_vnc_list.vnc.items
    WebJump           = data.sra_web_jump_list.web.items
  }
}
