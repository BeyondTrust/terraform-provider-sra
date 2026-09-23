package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The group policy membership endpoints interpolate a practitioner-supplied string
// into the request path. These cover the escaping that keeps that value inside a
// single path segment, so an ID is always an ID and never structure.
//
// The schema validators on group_policy_id (bt/rs/validators.go) are the primary
// constraint and reject everything exercised here; the escaping is what makes the
// path safe regardless of whether that constraint holds. Both layers are tested
// because either alone is a single point of failure.

func TestGroupPolicyEndpointsEscapeTheID(t *testing.T) {
	id := func(s string) *string { return &s }

	for _, tc := range []struct {
		name     string
		endpoint func(string) string
	}{
		{"vault-account-group", func(s string) string { return GroupPolicyVaultAccountGroup{GroupPolicyID: id(s)}.Endpoint() }},
		{"vault-account", func(s string) string { return GroupPolicyVaultAccount{GroupPolicyID: id(s)}.Endpoint() }},
		{"provision", func(s string) string { return GroupPolicyProvision{GroupPolicyID: id(s)}.Endpoint() }},
		{"jump-group", func(s string) string { return GroupPolicyJumpGroup{GroupPolicyID: id(s)}.Endpoint() }},
		{"jumpoint", func(s string) string { return GroupPolicyJumpoint{GroupPolicyID: id(s)}.Endpoint() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Conforming input is unchanged: escaping must not alter normal operation.
			assert.Equal(t, "group-policy/42/"+tc.name, tc.endpoint("42"),
				"a numeric id must pass through untouched")

			// A value containing separators stays one segment.
			got := tc.endpoint("../../admin")
			assert.NotContains(t, got, "../",
				"separators in the id must not survive as path structure: %s", got)
			assert.Equal(t, 3, len(strings.Split(got, "/")),
				"the path must still have exactly three segments: %s", got)
			assert.True(t, strings.HasSuffix(got, "/"+tc.name),
				"the trailing segment must remain intact: %s", got)
		})
	}
}

// TestEscapedEndpointSurvivesRequestConstruction confirms the escaping still holds
// once the endpoint becomes a real request.
//
// Assert on the WIRE form, not URL.Path. url.URL keeps two views of the path:
// Path is decoded for convenience and still reads `../../admin`, while RawPath /
// EscapedPath / RequestURI keep `..%2F..%2F` — and RequestURI is what is actually
// transmitted. Checking Path here would fail against correct behaviour and invite
// someone to "fix" it by removing the escaping.
//
// The unescaped comparison is included so this test demonstrates the escaping is
// doing the work, rather than asserting a property the URL type provides anyway:
// Go resolves dot-segments only in ResolveReference, never in Parse, so without
// the escape the value reaches the server as written.
func TestEscapedEndpointSurvivesRequestConstruction(t *testing.T) {
	const withSeparators = "../../admin"

	gpID := withSeparators
	escaped := GroupPolicyVaultAccount{GroupPolicyID: &gpID}.Endpoint()
	req, err := http.NewRequest(http.MethodGet, "https://appliance.example/api/config/v1/"+escaped, nil)
	require.NoError(t, err)

	assert.NotContains(t, req.URL.RequestURI(), "../",
		"the id must not survive as path structure: %s", req.URL.RequestURI())
	assert.Contains(t, req.URL.RequestURI(), "..%2F..%2Fadmin",
		"the id must travel as a single escaped segment: %s", req.URL.RequestURI())
	assert.Contains(t, req.URL.RequestURI(), "/api/config/v1/group-policy/",
		"the endpoint must still be rooted where it belongs: %s", req.URL.RequestURI())

	// Without the escape the separators survive to the wire. This is the behaviour
	// the escaping removes.
	unescaped := "group-policy/" + withSeparators + "/vault-account"
	bad, err := http.NewRequest(http.MethodGet, "https://appliance.example/api/config/v1/"+unescaped, nil)
	require.NoError(t, err)
	assert.Contains(t, bad.URL.RequestURI(), withSeparators,
		"sanity: an unescaped id reaches the wire intact, so the escape is load-bearing")
}
