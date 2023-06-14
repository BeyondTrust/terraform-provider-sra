package api

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

type testTFModel struct {
	ID types.String

	StringVal types.String
	IntVal    types.Int64
	BoolVal   types.Bool

	PointerStringVal types.String
	PointerIntVal    types.Int64
	PointerBoolVal   types.Bool

	NullStringVal types.String
	NullIntVal    types.Int64
	NullBoolVal   types.Bool

	UnknownStringVal types.String
	UnknownIntVal    types.Int64
	UnknownBoolVal   types.Bool

	ProductField      types.String `sraproduct:"rs"`
	APISkipField      types.String
	PersistStateField types.String `sra:"persist_state"`
	TFOnlyField       types.Bool
}

type testAPIModel struct {
	ID *int

	StringVal string
	IntVal    int
	BoolVal   bool

	PointerStringVal *string
	PointerIntVal    *int
	PointerBoolVal   *bool

	NullStringVal *string
	NullIntVal    *int
	NullBoolVal   *bool

	UnknownStringVal *string
	UnknownIntVal    *int
	UnknownBoolVal   *bool

	ProductField      *string
	APISkipField      *string `sraapi:"skip"`
	PersistStateField *string
}

func TestCopyTFtoAPI(t *testing.T) {
	// t.Parallel()

	tfObj := &testTFModel{
		ID:                types.StringValue("1"),
		StringVal:         types.StringValue("a string of some sort"),
		IntVal:            types.Int64Value(24601),
		BoolVal:           types.BoolValue(true),
		PointerStringVal:  types.StringValue("a different string"),
		PointerIntVal:     types.Int64Value(10642),
		PointerBoolVal:    types.BoolValue(false),
		NullStringVal:     types.StringNull(),
		NullIntVal:        types.Int64Null(),
		NullBoolVal:       types.BoolNull(),
		UnknownStringVal:  types.StringUnknown(),
		UnknownIntVal:     types.Int64Unknown(),
		UnknownBoolVal:    types.BoolUnknown(),
		ProductField:      types.StringValue("some product field"),
		PersistStateField: types.StringValue("state field"),
		APISkipField:      types.StringValue("api field value"),
		TFOnlyField:       types.BoolValue(false),
	}

	ctx := context.Background()

	tfElem := reflect.ValueOf(tfObj).Elem()

	for _, isRS := range []bool{false, true} {
		SetProductIsRS(isRS)

		var apiObj testAPIModel
		apiElem := reflect.ValueOf(&apiObj).Elem()
		CopyTFtoAPI(ctx, tfElem, apiElem)

		id, _ := strconv.Atoi(tfObj.ID.ValueString())
		assert.Equal(t, id, *apiObj.ID)

		assert.Equal(t, tfObj.StringVal.ValueString(), apiObj.StringVal)
		assert.Equal(t, int(tfObj.IntVal.ValueInt64()), apiObj.IntVal)
		assert.Equal(t, tfObj.BoolVal.ValueBool(), apiObj.BoolVal)

		assert.NotNil(t, apiObj.PointerStringVal)
		assert.NotNil(t, apiObj.PointerIntVal)
		assert.NotNil(t, apiObj.PointerBoolVal)
		assert.Equal(t, tfObj.PointerStringVal.ValueString(), *apiObj.PointerStringVal)
		assert.Equal(t, int(tfObj.PointerIntVal.ValueInt64()), *apiObj.PointerIntVal)
		assert.Equal(t, tfObj.PointerBoolVal.ValueBool(), *apiObj.PointerBoolVal)

		assert.Nil(t, apiObj.NullStringVal)
		assert.Nil(t, apiObj.NullIntVal)
		assert.Nil(t, apiObj.NullBoolVal)
		assert.Nil(t, apiObj.UnknownStringVal)
		assert.Nil(t, apiObj.UnknownIntVal)
		assert.Nil(t, apiObj.UnknownBoolVal)

		assert.Nil(t, apiObj.APISkipField)
		assert.Equal(t, tfObj.PersistStateField.ValueString(), *apiObj.PersistStateField)

		if isRS {
			assert.NotNil(t, apiObj.ProductField)
			if apiObj.ProductField != nil {
				assert.Equal(t, tfObj.ProductField.ValueString(), *apiObj.ProductField)
			}
		} else {
			assert.Nil(t, apiObj.ProductField)
		}
	}
}

func TestCopyAPItoTF(t *testing.T) {
	// t.Parallel()

	id := 1
	pointerString := "a different string"
	pointerInt := 10642
	pointerBool := false
	prodField := "some product field"
	apiField := "api field value"
	stateField := "stateField"
	persistedStateValue := "not what's in API"
	apiObj := &testAPIModel{
		ID:                &id,
		StringVal:         "a string of some sort",
		IntVal:            24601,
		BoolVal:           true,
		PointerStringVal:  &pointerString,
		PointerIntVal:     &pointerInt,
		PointerBoolVal:    &pointerBool,
		NullStringVal:     nil,
		NullIntVal:        nil,
		NullBoolVal:       nil,
		UnknownStringVal:  nil,
		UnknownIntVal:     nil,
		UnknownBoolVal:    nil,
		ProductField:      &prodField,
		APISkipField:      &apiField,
		PersistStateField: &stateField,
	}

	ctx := context.Background()

	apiElem := reflect.ValueOf(apiObj).Elem()
	apiType := reflect.TypeOf(apiObj).Elem()

	for _, isRS := range []bool{false, true} {
		SetProductIsRS(isRS)

		tfObj := &testTFModel{
			ID:                types.StringUnknown(),
			StringVal:         types.StringUnknown(),
			IntVal:            types.Int64Unknown(),
			BoolVal:           types.BoolUnknown(),
			PointerStringVal:  types.StringUnknown(),
			PointerIntVal:     types.Int64Unknown(),
			PointerBoolVal:    types.BoolUnknown(),
			NullStringVal:     types.StringNull(),
			NullIntVal:        types.Int64Null(),
			NullBoolVal:       types.BoolNull(),
			UnknownStringVal:  types.StringUnknown(),
			UnknownIntVal:     types.Int64Unknown(),
			UnknownBoolVal:    types.BoolUnknown(),
			ProductField:      types.StringUnknown(),
			PersistStateField: types.StringValue(persistedStateValue),
			APISkipField:      types.StringUnknown(),
			TFOnlyField:       types.BoolValue(false),
		}
		tfElem := reflect.ValueOf(tfObj).Elem()

		CopyAPItoTF(ctx, apiElem, tfElem, apiType)

		assert.Equal(t, strconv.Itoa(id), tfObj.ID.ValueString())

		assert.Equal(t, apiObj.StringVal, tfObj.StringVal.ValueString())
		assert.Equal(t, apiObj.IntVal, int(tfObj.IntVal.ValueInt64()))
		assert.Equal(t, apiObj.BoolVal, tfObj.BoolVal.ValueBool())

		assert.Equal(t, *apiObj.PointerStringVal, tfObj.PointerStringVal.ValueString())
		assert.Equal(t, *apiObj.PointerIntVal, int(tfObj.PointerIntVal.ValueInt64()))
		assert.Equal(t, *apiObj.PointerBoolVal, tfObj.PointerBoolVal.ValueBool())

		assert.True(t, tfObj.NullStringVal.IsNull())
		assert.True(t, tfObj.NullIntVal.IsNull())
		assert.True(t, tfObj.NullBoolVal.IsNull())

		assert.True(t, tfObj.NullStringVal.IsNull())
		assert.True(t, tfObj.NullIntVal.IsNull())
		assert.True(t, tfObj.NullBoolVal.IsNull())

		assert.True(t, tfObj.APISkipField.IsUnknown())
		assert.Equal(t, persistedStateValue, tfObj.PersistStateField.ValueString())

		assert.False(t, tfObj.TFOnlyField.ValueBool())

		if isRS {
			assert.Equal(t, tfObj.ProductField.ValueString(), *apiObj.ProductField)
		} else {
			assert.True(t, tfObj.ProductField.IsNull())
		}
	}
}
