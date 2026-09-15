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
// API client, in either of the two shapes these values actually appear in: a
// marshalled JSON body, and a Go struct rendered by fmt with %v or %+v. The second
// matters as much as the first — the call sites removed here included
// `%+v` of the item and of the Terraform plan, so that is the shape most likely to
// come back.
//
// Scope, stated precisely because it is narrower than it looks. Three limits:
//
//  1. tflog masking is carried on a context. Wiring it at provider configuration
//     (bt/provider.go) covers the API client, which logs through the context
//     captured there, and the remainder of Configure. It does NOT cover tflog
//     calls in the resource and data source handlers, which receive a fresh
//     context per RPC from the framework.
//  2. Only string values are masked. tflog applies these patterns to the message
//     and to string-valued fields; a struct handed over as a field value
//     (map[string]any{"item": item}) is passed through and rendered by the log
//     encoder untouched. Use logItem in bt/rs to log an item — it records the
//     type and nothing else — rather than passing the item itself.
//  3. These patterns cover the two shapes credentials actually appear in here:
//     marshalled JSON, and a struct rendered by fmt. Neither is a general
//     sanitiser.
//
// The call sites are kept safe by not logging payloads in the first place, which
// is the actual fix; this is a second line for the client's own output.

// sensitiveJSONKeys are the JSON object keys whose values are credentials.
//
// sensitiveGoFields are the corresponding Go struct field names, needed because
// fmt renders a struct as `Password:value` with no quotes or JSON punctuation, so
// the JSON patterns below cannot match it.
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

// Longest first: `PrivateKey` would otherwise be tried against
// `PrivateKeyPassphrase:` and fail on the required colon, which is harmless here
// but makes the intent of the ordering clearer to keep.
var sensitiveGoFields = []string{
	"PrivateKeyPassphrase",
	"PrivateKey",
	"Password",
	"Token",
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
	patterns := make([]*regexp.Regexp, 0, len(sensitiveJSONKeys)+len(sensitiveGoFields))

	for _, key := range sensitiveJSONKeys {
		patterns = append(patterns, regexp.MustCompile(
			fmt.Sprintf(`"%s"\s*:\s*"(?:\\.|[^"\\])*"`, regexp.QuoteMeta(key)),
		))
	}

	// Go struct rendering: `Password:hunter2`, terminated by whitespace or the
	// closing brace fmt puts at the end of a struct. Deliberately not \S+, which
	// would swallow a trailing `}` and with it the rest of a nested render.
	for _, field := range sensitiveGoFields {
		patterns = append(patterns, regexp.MustCompile(
			fmt.Sprintf(`%s:[^\s}]+`, regexp.QuoteMeta(field)),
		))
	}

	return patterns
}
