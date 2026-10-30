package bt

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

// The published type names are derived from the Go API type names
// (api.ToSnakeCase), so a type named MongoDBTunnelJump would quietly publish as
// sra_mongo_db_tunnel_jump, the way MySQLTunnelJump publishes as
// sra_my_sql_tunnel_jump. This pins the names through the provider's own
// registration, which is what Terraform sees.
func TestPublishedTypeNames(t *testing.T) {
	ctx := context.Background()
	p := New()

	var meta provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &meta)

	resources := map[string]bool{}
	for _, f := range p.Resources(ctx) {
		var resp resource.MetadataResponse
		f().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: meta.TypeName}, &resp)
		resources[resp.TypeName] = true
	}
	dataSources := map[string]bool{}
	for _, f := range p.DataSources(ctx) {
		var resp datasource.MetadataResponse
		f().Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: meta.TypeName}, &resp)
		dataSources[resp.TypeName] = true
	}

	for _, name := range []string{
		"sra_mongodb_tunnel_jump",
		"sra_oracle_tunnel_jump",
	} {
		assert.True(t, resources[name], "resource %s is not registered", name)
		assert.True(t, dataSources[name+"_list"], "data source %s_list is not registered", name)
	}
}
