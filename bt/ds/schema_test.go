package ds

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Everything under a read-only attribute (a list data source's items) is
// read-only too. Marking a nested attribute Required or Optional there changes
// nothing at runtime, since the parent can't be configured, but the generated
// docs list it under "Required" or "Optional" as though it could be set.
//
// Parents that are Optional+Computed are skipped: their children can be
// configured (the singular sra_single_vault_ssh_account's account block).
func TestReadOnlyAttributesHaveReadOnlyChildren(t *testing.T) {
	ctx := context.Background()
	for _, newDataSource := range DatasourceList() {
		d := newDataSource()
		var meta datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "sra"}, &meta)
		var resp datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &resp)
		require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

		for name, attr := range resp.Schema.Attributes {
			if attr.IsComputed() && !attr.IsOptional() {
				assertReadOnlyChildren(t, meta.TypeName+"."+name, attr)
			}
		}
	}
}

func assertReadOnlyChildren(t *testing.T, at string, attr schema.Attribute) {
	t.Helper()
	var children map[string]schema.Attribute
	switch a := attr.(type) {
	case schema.ListNestedAttribute:
		children = a.NestedObject.Attributes
	case schema.SetNestedAttribute:
		children = a.NestedObject.Attributes
	case schema.MapNestedAttribute:
		children = a.NestedObject.Attributes
	case schema.SingleNestedAttribute:
		children = a.Attributes
	}
	for name, child := range children {
		path := at + "." + name
		assert.True(t, child.IsComputed() && !child.IsRequired() && !child.IsOptional(),
			"%s is under a read-only attribute and must be Computed only", path)
		assertReadOnlyChildren(t, path, child)
	}
}
