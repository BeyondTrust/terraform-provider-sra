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
resource "sra_shell_jump" "shell_test" {
    name = "fun_jump"
    jumpoint_id = 1
    hostname = "test.host"
    protocol = "ssh"
    jump_group_id = 23
    jump_group_type = "personal"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "name", "fun_jump"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "jumpoint_id", "1"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "hostname", "test.host"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "protocol", "ssh"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "jump_group_id", "23"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "jump_group_type", "personal"),

					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "port", "22"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "terminal", "xterm"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "keep_alive", "0"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "tag", ""),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "comments", ""),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "username", ""),

					resource.TestCheckNoResourceAttr("sra_shell_jump.shell_test", "jump_policy_id"),
					resource.TestCheckNoResourceAttr("sra_shell_jump.shell_test", "session_policy_id"),

					resource.TestCheckResourceAttrSet("sra_shell_jump.shell_test", "id"),
				),
			},
			{
				ResourceName:      "sra_shell_jump.shell_test",
				ImportState:       true,
				ImportStateVerify: true,
			}, {
				Config: providerConfig + `
resource "sra_shell_jump" "shell_test" {
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
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "name", "fun_jump"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "jumpoint_id", "1"),
					// This has…
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "hostname", "test.host.changed"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "protocol", "ssh"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "jump_group_id", "23"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "jump_group_type", "personal"),

					// … and this has…
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "keep_alive", "100"),

					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "port", "22"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "terminal", "xterm"),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "tag", ""),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "comments", ""),
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "username", ""),

					resource.TestCheckNoResourceAttr("sra_shell_jump.shell_test", "jump_policy_id"),

					// … and this should now be set
					resource.TestCheckResourceAttr("sra_shell_jump.shell_test", "session_policy_id", "2"),

					resource.TestCheckResourceAttrSet("sra_shell_jump.shell_test", "id"),
				),
			},
		},
	})
}
