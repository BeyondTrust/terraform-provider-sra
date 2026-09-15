package test

import (
	"encoding/json"
	"testing"

	"terraform-provider-sra/api"

	"github.com/stretchr/testify/assert"
)

// The orphan guard is the only code in this package that deletes objects from the
// appliance, and its branches run solely when a `terraform import` fails -- so a
// green E2E run never executes them. These tests exercise the safety rule directly.
// They need no credentials and no appliance, and they run in the same invocation
// as the E2E suite, so the rule is checked on every run rather than only when
// something has already gone wrong.
//
// stageSkipRandomBits is declared in setup.go beside the code that substitutes it,
// so the guard and that substitution cannot drift apart -- which is why there is no
// test pinning the two together.

func TestMarkerUsable(t *testing.T) {
	for _, tc := range []struct {
		name       string
		randomBits string
		want       bool
		why        string
	}{
		{"a real marker", "abc123", true, "the normal case: setEnvAndGetRandom's lowercased UniqueId"},
		{"empty", "", false, "strings.Contains(x, \"\") is always true, which would disable the ownership check entirely"},
		{"stage-skip constant", stageSkipRandomBits, false, "shared by every test in a SKIP_ run, so it matches objects earlier runs left behind"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, markerUsable(tc.randomBits), tc.why)
		})
	}
}

func TestSafeToReclaim(t *testing.T) {
	name := func(s string) []byte {
		b, err := json.Marshal(api.VaultAccountGroup{Name: s})
		assert.NoError(t, err)
		return b
	}

	for _, tc := range []struct {
		name       string
		blob       []byte
		randomBits string
		want       bool
		why        string
	}{
		{
			"object carries the marker",
			name("This is a Name abc123 JIA"), "abc123", true,
			"the reclaim case: the object is demonstrably this run's",
		},
		{
			"object belongs to another run",
			name("This is a Name zzzzzz JIA"), "abc123", false,
			"refusing here is what prevents deleting a live object on a shared appliance",
		},
		{
			"empty marker against any object",
			name("This is a Name abc123 JIA"), "", false,
			"must not fall through to Contains, which would return true for everything",
		},
		{
			"stage-skip marker against a matching object",
			name("This is a Name not_so_random JIA"), stageSkipRandomBits, false,
			"the object does carry the marker, but the marker is shared -- it proves nothing",
		},
		{
			"zero-value object",
			name(""), "abc123", false,
			"a GET that somehow yields an empty object must not pass vacuously",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, safeToReclaim(tc.blob, tc.randomBits), tc.why)
		})
	}
}

// TestSafeToReclaimCoversWrongTypeNamespace is the scenario the ownership check
// exists for, expressed concretely: the guard's type parameter and its resource
// address are supplied independently, so a mis-paired case reads a same-numbered
// object from a different endpoint. jump-group 42 and vault/account-group 42 are
// unrelated objects. Whatever comes back will not carry this run's marker, because
// this run did not create it.
func TestSafeToReclaimCoversWrongTypeNamespace(t *testing.T) {
	somebodyElses, err := json.Marshal(api.JumpGroup{Name: "Production Jump Group", CodeName: "prod"})
	assert.NoError(t, err)

	assert.False(t, safeToReclaim(somebodyElses, "abc123"),
		"an object fetched from the wrong id namespace must never be deleted")
}
