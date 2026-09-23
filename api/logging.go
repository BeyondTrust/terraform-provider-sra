package api

import (
	"fmt"
	"regexp"
)

// Request and response bodies are not logged. Several attributes this provider
// manages are write-only (`password`, `private_key`, `private_key_passphrase`,
// `token`), and a body dump puts their values into debug output, which is shared
// far more readily than the values themselves should be. Log method, URL, status
// and size instead — that is the part with diagnostic value.
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
//  3. These patterns cover marshalled JSON, and a struct rendered by fmt with
//     %+v. Two shapes they cannot cover, both verified:
//
//     %v — without the plus, fmt omits field names entirely
//     ({someone hunter2 keep-me}), so there is nothing for a Field: pattern to
//     anchor on and no pattern can ever match. The four plan and state call
//     sites removed from bt/rs/api_resource.go used %v. Not logging the struct
//     is the only defence, which is why logItem exists and takes no format
//     string.
//
//     An unquoted %+v value containing whitespace — fmt separates fields with a
//     space, so `Password:two words Keep:x` is genuinely ambiguous and the match
//     stops at the first space. The quoted form (types.String, and therefore a
//     Terraform plan) is handled; a plain Go string field holding a password with
//     a space is not fully redacted.
//
//     Neither is a general sanitiser. The rule that call sites do not log
//     payloads is the fix; this is a net under it, with holes of a known size.
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

	// Go struct rendering under %+v: `Password:hunter2`, or `Password:"hunter2"`
	// when the field is a types.String (which has a String method), as in a
	// Terraform plan. The quoted alternative is tried first so a value containing
	// spaces or escaped quotes is consumed whole; the unquoted alternative stops at
	// whitespace or the closing brace, which is the best available guess when fmt
	// gives no delimiter at all.
	for _, field := range sensitiveGoFields {
		patterns = append(patterns, regexp.MustCompile(
			fmt.Sprintf(`%s:(?:"(?:\\.|[^"\\])*"|[^\s}]*)`, regexp.QuoteMeta(field)),
		))
	}

	return patterns
}
