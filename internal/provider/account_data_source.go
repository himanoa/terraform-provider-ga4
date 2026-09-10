package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	analyticsadmin "google.golang.org/api/analyticsadmin/v1beta"
)

var (
	_ datasource.DataSource              = (*accountDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*accountDataSource)(nil)
)

// accountDataSource は表示名から GA4 アカウントを引く。
// アカウントは API で作れない（管理画面でしか作れない）ので、データソースだけ用意する。
type accountDataSource struct {
	svc *analyticsadmin.Service
}

type accountDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	RegionCode  types.String `tfsdk:"region_code"`
}

func newAccountDataSource() datasource.DataSource {
	return &accountDataSource{}
}

func (d *accountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (d *accountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "表示名で GA4 アカウントを引く。`ga4_property` の `account` に渡すためのもの。",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "アカウントのリソース名（`accounts/123`）。",
			},
			"display_name": schema.StringAttribute{
				Required:    true,
				Description: "アカウントの表示名。完全一致で 1 件に絞れる必要がある。",
			},
			"region_code": schema.StringAttribute{
				Computed:    true,
				Description: "アカウントの国コード（例: `JP`）。",
			},
		},
	}
}

func (d *accountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if svc := serviceFromDataSourceConfigure(req, &resp.Diagnostics); svc != nil {
		d.svc = svc
	}
}

func (d *accountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config accountDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	displayName := config.DisplayName.ValueString()

	var matched []*analyticsadmin.GoogleAnalyticsAdminV1betaAccount

	err := d.svc.Accounts.List().Pages(ctx, func(page *analyticsadmin.GoogleAnalyticsAdminV1betaListAccountsResponse) error {
		for _, account := range page.Accounts {
			if account.DisplayName == displayName {
				matched = append(matched, account)
			}
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("GA4 アカウントの一覧を取得できません", err.Error())
		return
	}

	switch len(matched) {
	case 0:
		resp.Diagnostics.AddError(
			"GA4 アカウントが見つかりません",
			fmt.Sprintf("表示名 %q のアカウントが無いか、使っている資格情報にそのアカウントへの権限がありません。", displayName),
		)
		return
	case 1:
		// 期待どおり
	default:
		resp.Diagnostics.AddError(
			"GA4 アカウントが 1 件に絞れません",
			fmt.Sprintf("表示名 %q のアカウントが %d 件あります。管理画面で名前を変えて区別してください。", displayName, len(matched)),
		)
		return
	}

	account := matched[0]
	state := accountDataSourceModel{
		ID:          types.StringValue(account.Name),
		DisplayName: types.StringValue(account.DisplayName),
		RegionCode:  types.StringValue(account.RegionCode),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
