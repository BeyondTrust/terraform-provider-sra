package bt_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// Exercise Terraform's real state persistence after a failed Create, not just
// the framework response. The appliance is mocked; no live credentials are used.
func TestGroupPolicyMemberCLIRecoversAcceptedCreate(t *testing.T) {
	if os.Getenv("SRA_CLI_TEST") != "1" {
		t.Skip("set SRA_CLI_TEST=1 to run the local Terraform CLI regression")
	}
	terraform, err := exec.LookPath("terraform")
	require.NoError(t, err)
	dir := t.TempDir()
	build := exec.Command("go", "build", "-o", filepath.Join(dir, "terraform-provider-sra"), "..")
	output, err := build.CombinedOutput()
	require.NoError(t, err, "%s", output)
	cliConfig := filepath.Join(dir, "terraformrc")
	require.NoError(t, os.WriteFile(cliConfig, []byte(fmt.Sprintf(`provider_installation {
  dev_overrides { "beyondtrust/sra" = %q }
  direct {}
}`, dir)), 0600))
	for _, failure := range []string{"discovery", "provision"} {
		t.Run(failure, func(t *testing.T) {
			var failing atomic.Bool
			failing.Store(true)
			var creates atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case req.URL.Path == "/oauth2/token":
					_, _ = w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"test-token"}`))
				case strings.HasSuffix(req.URL.Path, "/get_mech_list"):
					_, _ = w.Write([]byte(`{}`))
				case req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/member"):
					creates.Add(1)
					w.WriteHeader(http.StatusCreated)
				case req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/member"):
					if failing.Load() && failure == "discovery" {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					_, _ = w.Write([]byte(`[{"id":77,"security_provider_id":5,"group_name":"Suppliers"}]`))
				case strings.HasSuffix(req.URL.Path, "/member/77"):
					_, _ = w.Write([]byte(`{"id":77,"security_provider_id":5,"group_name":"Suppliers"}`))
				case strings.HasSuffix(req.URL.Path, "/provision"):
					w.WriteHeader(http.StatusInternalServerError)
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			work := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(work, "main.tf"), []byte(fmt.Sprintf(`terraform {
  required_providers { sra = { source = "beyondtrust/sra" } }
}
provider "sra" {
  host = %q
  client_id = "test"
  client_secret = "test"
}
resource "sra_group_policy_member" "test" {
  group_policy_id = "9"
  security_provider_id = 5
  group_name = "Suppliers"
}`, server.URL)), 0600))
			run := func(args ...string) ([]byte, error) {
				cmd := exec.Command(terraform, args...)
				cmd.Dir = work
				cmd.Env = append(os.Environ(), "TF_CLI_CONFIG_FILE="+cliConfig, "TF_IN_AUTOMATION=1")
				return cmd.CombinedOutput()
			}
			output, err := run("apply", "-auto-approve", "-input=false", "-no-color")
			require.Error(t, err, "%s", output)
			state, err := os.ReadFile(filepath.Join(work, "terraform.tfstate"))
			require.NoError(t, err, "%s", output)
			var saved struct {
				Resources []struct {
					Instances []struct {
						Attributes map[string]any `json:"attributes"`
					} `json:"instances"`
				} `json:"resources"`
			}
			require.NoError(t, json.Unmarshal(state, &saved))
			require.Len(t, saved.Resources, 1, "%s", output)
			require.Len(t, saved.Resources[0].Instances, 1)
			require.Equal(t, "Suppliers", saved.Resources[0].Instances[0].Attributes["group_name"])
			failing.Store(false)
			output, err = run("apply", "-refresh-only", "-auto-approve", "-input=false", "-no-color")
			require.NoError(t, err, "%s", output)
			state, err = os.ReadFile(filepath.Join(work, "terraform.tfstate"))
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(state, &saved))
			require.Equal(t, "77", saved.Resources[0].Instances[0].Attributes["id"])
			require.Equal(t, int32(1), creates.Load())
		})
	}
}
