package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimestampJson(t *testing.T) {
	t.Parallel()

	type tsTest struct {
		Field Timestamp
	}

	// Timestamp is always UTC
	test := &tsTest{"1983-01-24T23:47:00Z"}
	testJson := []byte(`{"Field":412300020}`)

	result, err := json.Marshal(test)
	assert.Nil(t, err)
	assert.Equal(t, testJson, result)

	var output tsTest
	err = json.Unmarshal(testJson, &output)
	assert.Nil(t, err)
	assert.Equal(t, test.Field, output.Field)
}

func TestConfigBool(t *testing.T) {
	t.Parallel()

	type boolTest struct {
		Field ConfigBool
	}

	trueTest := &boolTest{true}
	trueResult := []byte(`{"Field":"1"}`)
	falseTest := &boolTest{false}
	falseResult := []byte(`{"Field":"0"}`)

	result, err := json.Marshal(trueTest)
	assert.Nil(t, err)
	assert.Equal(t, trueResult, result)

	result, err = json.Marshal(falseTest)
	assert.Nil(t, err)
	assert.Equal(t, falseResult, result)

	var unmarshalTest boolTest
	err = json.Unmarshal(trueResult, &unmarshalTest)
	assert.Nil(t, err)
	assert.Equal(t, trueTest.Field, unmarshalTest.Field)

	err = json.Unmarshal(falseResult, &unmarshalTest)
	assert.Nil(t, err)
	assert.Equal(t, falseTest.Field, unmarshalTest.Field)
}
