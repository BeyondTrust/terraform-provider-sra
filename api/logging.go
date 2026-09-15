package api

import (
	"fmt"
	"regexp"
)

// Request and response bodies are not logged. Several attributes this provider
// manages are write-only credentials (`password`, `private_key`,
// `private_key_passphrase`, `token`), and a body dump puts them verbatim into the
// Terraform debug log, which routinely gets attached to support tickets. Log
// method, URL, status and size instead — that is the part with diagnostic value.
//
// The helpers here are the backstop for anything that slips past that rule in the
// API client: a new log call, or a message assembled with fmt.Sprintf.
//
// Scope, stated precisely because it is narrower than it looks: tflog masking is
// carried on a context, so wiring it at provider configuration (bt/provider.go)
// covers the API client — which logs through the logCtx captured there — and the
// remainder of Configure. It does NOT cover tflog calls in the resource and data
// source handlers, which receive a fresh context per RPC from the framework. Those
// call sites are kept safe by not logging payloads in the first place, which is the
// actual fix; this is a second line for the client's own output.

// sensitiveJSONKeys are the JSON object keys whose values are credentials.
//
// Keep in sync with the write-only attributes in models.go: VaultUsernamePassword's
// Password, VaultSSHAccount's PrivateKey and PrivateKeyPassphrase, and
// VaultTokenAccount's Token.
//
// TestSensitiveValuePatternsRedactEveryCredentialField marshals those models with a
// canary in each field and fails if one survives, so a renamed json tag is caught.
// A credential added to a model NOT in that test's list is not caught — add it there.
var sensitiveJSONKeys = []string{
	"password",
	"private_key",
	"private_key_passphrase",
	"token",
}

// SensitiveValuePatterns returns one regex per sensitive key, each matching a whole
// JSON key/value pair so the value cannot survive in the output.
//
// tflog replaces the entire match with "***", so matching the pair rather than just
// the value is deliberate: a pattern that matched only the value would need a
// lookbehind for the key, which Go's regexp does not support.
//
// The value pattern tolerates escaped quotes (`\"`) inside the string, so a
// credential containing a quote cannot terminate the match early and leave its tail
// exposed.
func SensitiveValuePatterns() []*regexp.Regexp {
	patterns := make([]*regexp.Regexp, 0, len(sensitiveJSONKeys))
	for _, key := range sensitiveJSONKeys {
		patterns = append(patterns, regexp.MustCompile(
			fmt.Sprintf(`"%s"\s*:\s*"(?:\\.|[^"\\])*"`, regexp.QuoteMeta(key)),
		))
	}
	return patterns
}
