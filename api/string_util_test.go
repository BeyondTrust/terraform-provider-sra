package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransformToSnakeCase(t *testing.T) {
	t.Parallel()

	testString := ToSnakeCase("thisIsATestString123")
	assert.Equal(t, "this_is_a_test_string123", testString)

	testString = ToSnakeCase("ThisOneHASCaps")
	assert.Equal(t, "this_one_has_caps", testString)

}
