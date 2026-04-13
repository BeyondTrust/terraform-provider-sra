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

type testTFModelWithSet struct {
	ID        types.String
	Name      types.String
	EmailList types.Set `sraproduct:"pra"`
}

type testAPIModelWithSet struct {
	ID        *int
	Name      string
	EmailList *[]string `sraproduct:"pra"`
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
		product := ProductPRA
		if isRS {
			product = ProductRS
		}

		var apiObj testAPIModel
		apiElem := reflect.ValueOf(&apiObj).Elem()
		CopyTFtoAPI(ctx, tfElem, apiElem, product)

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
		product := ProductPRA
		if isRS {
			product = ProductRS
		}

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

		CopyAPItoTF(ctx, apiElem, tfElem, apiType, product)

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

func TestCopyAPItoTF_NilPointersSetToNull(t *testing.T) {
	ctx := context.Background()
	product := ProductPRA

	id := 42
	apiObj := &testAPIModel{
		ID:                &id,
		StringVal:         "hello",
		IntVal:            1,
		BoolVal:           true,
		PointerStringVal:  nil,
		PointerIntVal:     nil,
		PointerBoolVal:    nil,
		NullStringVal:     nil,
		NullIntVal:        nil,
		NullBoolVal:       nil,
		UnknownStringVal:  nil,
		UnknownIntVal:     nil,
		UnknownBoolVal:    nil,
		ProductField:      nil,
		PersistStateField: nil,
	}

	tfObj := &testTFModel{
		ID:                types.StringUnknown(),
		StringVal:         types.StringUnknown(),
		IntVal:            types.Int64Unknown(),
		BoolVal:           types.BoolUnknown(),
		PointerStringVal:  types.StringUnknown(),
		PointerIntVal:     types.Int64Unknown(),
		PointerBoolVal:    types.BoolUnknown(),
		NullStringVal:     types.StringUnknown(),
		NullIntVal:        types.Int64Unknown(),
		NullBoolVal:       types.BoolUnknown(),
		UnknownStringVal:  types.StringUnknown(),
		UnknownIntVal:     types.Int64Unknown(),
		UnknownBoolVal:    types.BoolUnknown(),
		ProductField:      types.StringUnknown(),
		PersistStateField: types.StringValue("keep me"),
		APISkipField:      types.StringValue("untouched"),
		TFOnlyField:       types.BoolValue(false),
	}

	apiElem := reflect.ValueOf(apiObj).Elem()
	apiType := reflect.TypeOf(apiObj).Elem()
	tfElem := reflect.ValueOf(tfObj).Elem()

	CopyAPItoTF(ctx, apiElem, tfElem, apiType, product)

	// Non-pointer fields should be set
	assert.Equal(t, "42", tfObj.ID.ValueString())
	assert.Equal(t, "hello", tfObj.StringVal.ValueString())
	assert.Equal(t, int64(1), tfObj.IntVal.ValueInt64())
	assert.Equal(t, true, tfObj.BoolVal.ValueBool())

	// Nil pointer fields should become TF null
	assert.True(t, tfObj.PointerStringVal.IsNull(), "nil *string should produce types.StringNull")
	assert.True(t, tfObj.PointerIntVal.IsNull(), "nil *int should produce types.Int64Null")
	assert.True(t, tfObj.PointerBoolVal.IsNull(), "nil *bool should produce types.BoolNull")

	assert.True(t, tfObj.NullStringVal.IsNull())
	assert.True(t, tfObj.NullIntVal.IsNull())
	assert.True(t, tfObj.NullBoolVal.IsNull())

	// persist_state field should be preserved, not overwritten
	assert.Equal(t, "keep me", tfObj.PersistStateField.ValueString())

	// sraapi:"skip" field should be untouched
	assert.Equal(t, "untouched", tfObj.APISkipField.ValueString())

	// nil PRA field in PRA mode should be null (nil pointer path, not product mismatch path)
	assert.True(t, tfObj.ProductField.IsNull())
}

func TestCopyTFtoAPI_EmptyStringPointerSkipped(t *testing.T) {
	ctx := context.Background()
	product := ProductPRA

	tfObj := &testTFModel{
		ID:                types.StringValue("1"),
		StringVal:         types.StringValue("hello"),
		IntVal:            types.Int64Value(1),
		BoolVal:           types.BoolValue(true),
		PointerStringVal:  types.StringValue(""), // empty string should be skipped for pointer fields
		PointerIntVal:     types.Int64Value(0),
		PointerBoolVal:    types.BoolValue(false),
		NullStringVal:     types.StringNull(),
		NullIntVal:        types.Int64Null(),
		NullBoolVal:       types.BoolNull(),
		UnknownStringVal:  types.StringUnknown(),
		UnknownIntVal:     types.Int64Unknown(),
		UnknownBoolVal:    types.BoolUnknown(),
		ProductField:      types.StringNull(),
		PersistStateField: types.StringValue("x"),
		APISkipField:      types.StringValue("x"),
		TFOnlyField:       types.BoolValue(false),
	}

	var apiObj testAPIModel
	CopyTFtoAPI(ctx, reflect.ValueOf(tfObj).Elem(), reflect.ValueOf(&apiObj).Elem(), product)

	assert.Nil(t, apiObj.PointerStringVal, "Empty string on a pointer field should be skipped (omitempty)")
}

func TestCopyTFtoAPI_NullID(t *testing.T) {
	ctx := context.Background()
	product := ProductPRA

	tfObj := &testTFModel{
		ID:                types.StringNull(),
		StringVal:         types.StringValue("hello"),
		IntVal:            types.Int64Value(1),
		BoolVal:           types.BoolValue(true),
		PointerStringVal:  types.StringNull(),
		PointerIntVal:     types.Int64Null(),
		PointerBoolVal:    types.BoolNull(),
		NullStringVal:     types.StringNull(),
		NullIntVal:        types.Int64Null(),
		NullBoolVal:       types.BoolNull(),
		UnknownStringVal:  types.StringUnknown(),
		UnknownIntVal:     types.Int64Unknown(),
		UnknownBoolVal:    types.BoolUnknown(),
		ProductField:      types.StringNull(),
		PersistStateField: types.StringValue("x"),
		APISkipField:      types.StringValue("x"),
		TFOnlyField:       types.BoolValue(false),
	}

	var apiObj testAPIModel
	CopyTFtoAPI(ctx, reflect.ValueOf(tfObj).Elem(), reflect.ValueOf(&apiObj).Elem(), product)

	assert.Nil(t, apiObj.ID, "Null TF ID should leave API ID nil")
	assert.Equal(t, "hello", apiObj.StringVal, "Other fields should still copy")
}

func TestCopyTFtoAPI_UnknownID(t *testing.T) {
	ctx := context.Background()
	product := ProductPRA

	tfObj := &testTFModel{
		ID:                types.StringUnknown(),
		StringVal:         types.StringValue("hello"),
		IntVal:            types.Int64Value(1),
		BoolVal:           types.BoolValue(true),
		PointerStringVal:  types.StringNull(),
		PointerIntVal:     types.Int64Null(),
		PointerBoolVal:    types.BoolNull(),
		NullStringVal:     types.StringNull(),
		NullIntVal:        types.Int64Null(),
		NullBoolVal:       types.BoolNull(),
		UnknownStringVal:  types.StringUnknown(),
		UnknownIntVal:     types.Int64Unknown(),
		UnknownBoolVal:    types.BoolUnknown(),
		ProductField:      types.StringNull(),
		PersistStateField: types.StringValue("x"),
		APISkipField:      types.StringValue("x"),
		TFOnlyField:       types.BoolValue(false),
	}

	var apiObj testAPIModel
	CopyTFtoAPI(ctx, reflect.ValueOf(tfObj).Elem(), reflect.ValueOf(&apiObj).Elem(), product)

	assert.Nil(t, apiObj.ID, "Unknown TF ID should leave API ID nil")
}

func TestCopyTFtoAPI_SliceField(t *testing.T) {
	ctx := context.Background()
	product := ProductPRA // PRA mode so sraproduct:"pra" fields are active

	emails := []string{"a@b.com", "c@d.com"}
	emailSet, _ := types.SetValueFrom(ctx, types.StringType, emails)

	tfObj := &testTFModelWithSet{
		ID:        types.StringValue("5"),
		Name:      types.StringValue("test"),
		EmailList: emailSet,
	}

	var apiObj testAPIModelWithSet
	CopyTFtoAPI(ctx, reflect.ValueOf(tfObj).Elem(), reflect.ValueOf(&apiObj).Elem(), product)

	assert.NotNil(t, apiObj.ID)
	assert.Equal(t, 5, *apiObj.ID)
	assert.Equal(t, "test", apiObj.Name)
	// The slice field is not currently handled by CopyTFtoAPI's switch — the Slice case
	// exists but has no active code. The pointer gets allocated (non-nil) during the
	// pointer-dereferencing step, but the underlying slice remains nil.
	// After refactor, this assertion should change to verify the slice is populated.
	assert.NotNil(t, apiObj.EmailList, "Current behavior: pointer is allocated but slice content is not populated")
	assert.Nil(t, *apiObj.EmailList, "Current behavior: underlying slice is nil (slice handling is a no-op)")
}

func TestCopyTFtoAPI_SliceFieldProductMismatch(t *testing.T) {
	ctx := context.Background()
	product := ProductRS // RS mode — sraproduct:"pra" field should be skipped

	tfObj := &testTFModelWithSet{
		ID:        types.StringValue("5"),
		Name:      types.StringValue("test"),
		EmailList: types.SetNull(types.StringType),
	}

	var apiObj testAPIModelWithSet
	CopyTFtoAPI(ctx, reflect.ValueOf(tfObj).Elem(), reflect.ValueOf(&apiObj).Elem(), product)

	assert.NotNil(t, apiObj.ID)
	assert.Equal(t, 5, *apiObj.ID)
	assert.Equal(t, "test", apiObj.Name)
	assert.Nil(t, apiObj.EmailList, "PRA field should be skipped in RS mode")
}

func TestCopyAPItoTF_SliceToSet(t *testing.T) {
	ctx := context.Background()
	product := ProductPRA // PRA mode

	id := 10
	emails := []string{"a@b.com", "c@d.com"}
	apiObj := &testAPIModelWithSet{
		ID:        &id,
		Name:      "test",
		EmailList: &emails,
	}

	tfObj := &testTFModelWithSet{
		ID:        types.StringUnknown(),
		Name:      types.StringUnknown(),
		EmailList: types.SetUnknown(types.StringType),
	}

	apiElem := reflect.ValueOf(apiObj).Elem()
	apiType := reflect.TypeOf(apiObj).Elem()
	tfElem := reflect.ValueOf(tfObj).Elem()

	CopyAPItoTF(ctx, apiElem, tfElem, apiType, product)

	assert.Equal(t, "10", tfObj.ID.ValueString())
	assert.Equal(t, "test", tfObj.Name.ValueString())

	assert.False(t, tfObj.EmailList.IsNull(), "Populated slice should not be null")
	assert.False(t, tfObj.EmailList.IsUnknown(), "Populated slice should not be unknown")

	var result []string
	diag := tfObj.EmailList.ElementsAs(ctx, &result, false)
	assert.False(t, diag.HasError())
	assert.ElementsMatch(t, []string{"a@b.com", "c@d.com"}, result)
}

func TestCopyAPItoTF_NilSliceToNullSet(t *testing.T) {
	ctx := context.Background()
	product := ProductPRA // PRA mode

	id := 10
	apiObj := &testAPIModelWithSet{
		ID:        &id,
		Name:      "test",
		EmailList: nil, // nil slice
	}

	tfObj := &testTFModelWithSet{
		ID:        types.StringUnknown(),
		Name:      types.StringUnknown(),
		EmailList: types.SetUnknown(types.StringType),
	}

	apiElem := reflect.ValueOf(apiObj).Elem()
	apiType := reflect.TypeOf(apiObj).Elem()
	tfElem := reflect.ValueOf(tfObj).Elem()

	CopyAPItoTF(ctx, apiElem, tfElem, apiType, product)

	assert.True(t, tfObj.EmailList.IsNull(), "Nil API slice should produce null TF set")
}

func TestCopyAPItoTF_EmptySliceToEmptySet(t *testing.T) {
	ctx := context.Background()
	product := ProductPRA

	id := 10
	emails := []string{} // empty, not nil
	apiObj := &testAPIModelWithSet{
		ID:        &id,
		Name:      "test",
		EmailList: &emails,
	}

	tfObj := &testTFModelWithSet{
		ID:        types.StringUnknown(),
		Name:      types.StringUnknown(),
		EmailList: types.SetUnknown(types.StringType),
	}

	apiElem := reflect.ValueOf(apiObj).Elem()
	apiType := reflect.TypeOf(apiObj).Elem()
	tfElem := reflect.ValueOf(tfObj).Elem()

	CopyAPItoTF(ctx, apiElem, tfElem, apiType, product)

	assert.False(t, tfObj.EmailList.IsNull(), "Empty slice should produce empty set, not null")
	assert.Equal(t, 0, len(tfObj.EmailList.Elements()))
}

func TestCopyAPItoTF_SliceProductMismatch(t *testing.T) {
	ctx := context.Background()
	product := ProductRS // RS mode — sraproduct:"pra" field should be nulled

	id := 10
	apiObj := &testAPIModelWithSet{
		ID:        &id,
		Name:      "test",
		EmailList: nil,
	}

	tfObj := &testTFModelWithSet{
		ID:        types.StringUnknown(),
		Name:      types.StringUnknown(),
		EmailList: types.SetUnknown(types.StringType),
	}

	apiElem := reflect.ValueOf(apiObj).Elem()
	apiType := reflect.TypeOf(apiObj).Elem()
	tfElem := reflect.ValueOf(tfObj).Elem()

	CopyAPItoTF(ctx, apiElem, tfElem, apiType, product)

	assert.True(t, tfObj.EmailList.IsNull(), "PRA slice field in RS mode should be null")
}
