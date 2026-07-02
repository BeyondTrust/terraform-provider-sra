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

	{
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

	{
		// Timestamp is always UTC
		test := &tsTest{"not a date"}
		testJson := []byte(`{"Field":"not a number"}`)

		result, err := json.Marshal(test)
		assert.NotNil(t, err)
		assert.Nil(t, result)

		var output tsTest
		err = json.Unmarshal(testJson, &output)
		assert.NotNil(t, err)
		assert.Empty(t, output.Field)
	}
}

func TestTimestampUnmarshal_NullAndEmpty(t *testing.T) {
	t.Parallel()

	type tsTest struct {
		Field Timestamp
	}

	// JSON null
	var output tsTest
	err := json.Unmarshal([]byte(`{"Field":null}`), &output)
	assert.Nil(t, err)
	assert.Equal(t, Timestamp(""), output.Field)

	// Empty string
	output = tsTest{}
	err = json.Unmarshal([]byte(`{"Field":""}`), &output)
	assert.Nil(t, err)
	assert.Equal(t, Timestamp(""), output.Field)
}

func TestTimestampUnmarshal_QuotedNumeric(t *testing.T) {
	t.Parallel()

	type tsTest struct {
		Field Timestamp
	}

	// Quoted numeric (unix seconds as string)
	var output tsTest
	err := json.Unmarshal([]byte(`{"Field":"412300020"}`), &output)
	assert.Nil(t, err)
	assert.Equal(t, Timestamp("1983-01-24T23:47:00Z"), output.Field)
}

func TestTimestampUnmarshal_UnquotedNumeric(t *testing.T) {
	t.Parallel()

	type tsTest struct {
		Field Timestamp
	}

	// Unquoted numeric (unix seconds as number)
	var output tsTest
	err := json.Unmarshal([]byte(`{"Field":412300020}`), &output)
	assert.Nil(t, err)
	assert.Equal(t, Timestamp("1983-01-24T23:47:00Z"), output.Field)
}

func TestTimestampUnmarshal_QuotedRFC3339(t *testing.T) {
	t.Parallel()

	type tsTest struct {
		Field Timestamp
	}

	var output tsTest
	err := json.Unmarshal([]byte(`{"Field":"2024-06-15T10:30:00Z"}`), &output)
	assert.Nil(t, err)
	assert.Equal(t, Timestamp("2024-06-15T10:30:00Z"), output.Field)
}

func TestTimestampUnmarshal_Epoch(t *testing.T) {
	t.Parallel()

	type tsTest struct {
		Field Timestamp
	}

	// Unix epoch as unquoted 0
	var output tsTest
	err := json.Unmarshal([]byte(`{"Field":0}`), &output)
	assert.Nil(t, err)
	assert.Equal(t, Timestamp("1970-01-01T00:00:00Z"), output.Field)
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
