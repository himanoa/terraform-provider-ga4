package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// スキーマ定義の整合性（Optional+Computed+Default の組み合わせ、Required に Default が無いか等）は
// framework 側の ValidateImplementation が検査する。API を叩かずに済むので通常のテストとして回す。

func TestProviderSchema(t *testing.T) {
	ctx := context.Background()
	p := New("test")()

	var resp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("provider schema: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("provider schema validate: %v", diags)
	}

	var meta provider.MetadataResponse
	p.Metadata(ctx, provider.MetadataRequest{}, &meta)
	if meta.TypeName != "ga4" {
		t.Errorf("TypeName = %q, want ga4", meta.TypeName)
	}
}

func TestResourceSchemas(t *testing.T) {
	ctx := context.Background()
	p := New("test")()

	want := map[string]bool{
		"ga4_property":                    false,
		"ga4_data_stream":                 false,
		"ga4_measurement_protocol_secret": false,
		"ga4_custom_dimension":            false,
	}

	for _, factory := range p.Resources(ctx) {
		r := factory()

		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "ga4"}, &meta)

		if _, ok := want[meta.TypeName]; !ok {
			t.Errorf("想定していないリソース %q", meta.TypeName)
			continue
		}
		want[meta.TypeName] = true

		var resp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("%s schema: %v", meta.TypeName, resp.Diagnostics)
			continue
		}
		if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("%s schema validate: %v", meta.TypeName, diags)
		}

		// import できることをリソース全部で保証する（既に手で作ったものを取り込む経路）
		if _, ok := r.(resource.ResourceWithImportState); !ok {
			t.Errorf("%s は ImportState を実装していない", meta.TypeName)
		}
	}

	for name, seen := range want {
		if !seen {
			t.Errorf("リソース %q が登録されていない", name)
		}
	}
}

func TestDataSourceSchemas(t *testing.T) {
	ctx := context.Background()
	p := New("test")()

	seen := map[string]bool{}

	for _, factory := range p.DataSources(ctx) {
		d := factory()

		var meta datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "ga4"}, &meta)
		seen[meta.TypeName] = true

		var resp datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("%s schema: %v", meta.TypeName, resp.Diagnostics)
			continue
		}
		if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("%s schema validate: %v", meta.TypeName, diags)
		}
	}

	if !seen["ga4_account"] {
		t.Errorf("データソース ga4_account が登録されていない")
	}
}
