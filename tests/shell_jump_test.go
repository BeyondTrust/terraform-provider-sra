package test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccShellJumpResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "bt_shell_jump" "shell_test" {
    name = "fun_jump"
    jumpoint_id = 1
    hostname = "test.host"
    protocol = "ssh"
    jump_group_id = 23
    jump_group_type = "personal"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "name", "fun_jump"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "jumpoint_id", "1"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "hostname", "test.host"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "protocol", "ssh"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "jump_group_id", "23"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "jump_group_type", "personal"),

					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "port", "22"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "terminal", "xterm"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "keep_alive", "0"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "tag", ""),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "comments", ""),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "username", ""),

					resource.TestCheckNoResourceAttr("bt_shell_jump.shell_test", "jump_policy_id"),
					resource.TestCheckNoResourceAttr("bt_shell_jump.shell_test", "session_policy_id"),

					resource.TestCheckResourceAttrSet("bt_shell_jump.shell_test", "id"),
				),
			},
			{
				ResourceName:	   "bt_shell_jump.shell_test",
				ImportState:	   true,
				ImportStateVerify: true,
			},{
				Config: providerConfig + `
resource "bt_shell_jump" "shell_test" {
    name = "fun_jump"
    jumpoint_id = 1
    hostname = "test.host.changed"
    protocol = "ssh"
    jump_group_id = 23
    jump_group_type = "personal"
	keep_alive = 100
	session_policy_id = 2
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Most of this hasn't changed
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "name", "fun_jump"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "jumpoint_id", "1"),
					// This has…
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "hostname", "test.host.changed"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "protocol", "ssh"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "jump_group_id", "23"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "jump_group_type", "personal"),

					// … and this has…
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "keep_alive", "100"),

					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "port", "22"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "terminal", "xterm"),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "tag", ""),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "comments", ""),
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "username", ""),

					resource.TestCheckNoResourceAttr("bt_shell_jump.shell_test", "jump_policy_id"),

					// … and this should now be set
					resource.TestCheckResourceAttr("bt_shell_jump.shell_test", "session_policy_id", "2"),

					resource.TestCheckResourceAttrSet("bt_shell_jump.shell_test", "id"),
				),
			},
		},
	})
}