package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	analyticsadmin "google.golang.org/api/analyticsadmin/v1beta"
)

var (
	_ resource.Resource                = (*propertyResource)(nil)
	_ resource.ResourceWithConfigure   = (*propertyResource)(nil)
	_ resource.ResourceWithImportState = (*propertyResource)(nil)
)

// propertyResource は GA4 プロパティ。
//
// destroy すると API の delete を呼ぶが、GA4 のプロパティ削除は「ゴミ箱に入れる」であり
// 35 日間は管理画面から復元できる。同じ表示名で作り直しても別のプロパティになる。
type propertyResource struct {
	svc *analyticsadmin.Service
}

type propertyResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Account          types.String `tfsdk:"account"`
	DisplayName      types.String `tfsdk:"display_name"`
	TimeZone         types.String `tfsdk:"time_zone"`
	CurrencyCode     types.String `tfsdk:"currency_code"`
	IndustryCategory types.String `tfsdk:"industry_category"`
	PropertyType     types.String `tfsdk:"property_type"`
	ServiceLevel     types.String `tfsdk:"service_level"`
}

func newPropertyResource() resource.Resource {
	return &propertyResource{}
}

func (r *propertyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_property"
}

func (r *propertyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "GA4 プロパティ。destroy はゴミ箱への移動で、35 日間は管理画面から復元できる。" +
			"レポート用識別子・データ保持期間など Admin API v1beta に無い設定は管理画面で行う。",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "プロパティのリソース名（`properties/123`）。子リソースの `property` にこれを渡す。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"account": schema.StringAttribute{
				Required:      true,
				Description:   "所属するアカウントのリソース名（`accounts/123`）。`data.ga4_account` の `id` を渡す。変更すると作り直しになる。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"display_name": schema.StringAttribute{
				Required:    true,
				Description: "プロパティの表示名。",
			},
			"time_zone": schema.StringAttribute{
				Required:    true,
				Description: "レポートのタイムゾーン（例: `Asia/Tokyo`）。",
			},
			"currency_code": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "レポートの通貨（例: `JPY`）。省略すると GA4 側の既定値になる。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"industry_category": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "業種（例: `GAMES`、`ONLINE_COMMUNITIES`）。省略すると未設定のまま。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"property_type": schema.StringAttribute{
				Computed:      true,
				Description:   "プロパティの種類。API で作るものは `PROPERTY_TYPE_ORDINARY`。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_level": schema.StringAttribute{
				Computed:      true,
				Description:   "`GOOGLE_ANALYTICS_STANDARD` か `GOOGLE_ANALYTICS_360`。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *propertyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if svc := serviceFromResourceConfigure(req, &resp.Diagnostics); svc != nil {
		r.svc = svc
	}
}

func (r *propertyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan propertyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.svc.Properties.Create(&analyticsadmin.GoogleAnalyticsAdminV1betaProperty{
		Parent:           plan.Account.ValueString(),
		DisplayName:      plan.DisplayName.ValueString(),
		TimeZone:         plan.TimeZone.ValueString(),
		CurrencyCode:     knownString(plan.CurrencyCode),
		IndustryCategory: knownString(plan.IndustryCategory),
	}).Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("GA4 プロパティを作成できません", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, propertyModelFromAPI(created))...)
}

func (r *propertyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state propertyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := r.svc.Properties.Get(state.ID.ValueString()).Context(ctx).Do()
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GA4 プロパティを取得できません", err.Error())
		return
	}

	// ゴミ箱に入っているプロパティは Get で返ってくるが、もう使えないので無いものとして扱う
	if got.DeleteTime != "" {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, propertyModelFromAPI(got))...)
}

func (r *propertyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state propertyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// 設定に書かれている（plan で値が決まっている）ものだけを更新対象にする。
	// currency_code / industry_category は省略可なので、書かれていないときは GA4 側の値を保つ
	body := &analyticsadmin.GoogleAnalyticsAdminV1betaProperty{
		DisplayName: plan.DisplayName.ValueString(),
		TimeZone:    plan.TimeZone.ValueString(),
	}
	mask := []string{"displayName", "timeZone"}

	if v := knownString(plan.CurrencyCode); v != "" {
		body.CurrencyCode = v
		mask = append(mask, "currencyCode")
	}
	if v := knownString(plan.IndustryCategory); v != "" {
		body.IndustryCategory = v
		mask = append(mask, "industryCategory")
	}

	updated, err := r.svc.Properties.Patch(state.ID.ValueString(), body).UpdateMask(strings.Join(mask, ",")).Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("GA4 プロパティを更新できません", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, propertyModelFromAPI(updated))...)
}

func (r *propertyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state propertyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.svc.Properties.Delete(state.ID.ValueString()).Context(ctx).Do(); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("GA4 プロパティを削除（ゴミ箱へ移動）できません", err.Error())
		return
	}
}

// ImportState は `terraform import ga4_property.x properties/123` を受ける
func (r *propertyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func propertyModelFromAPI(p *analyticsadmin.GoogleAnalyticsAdminV1betaProperty) *propertyResourceModel {
	return &propertyResourceModel{
		ID:               types.StringValue(p.Name),
		Account:          types.StringValue(p.Parent),
		DisplayName:      types.StringValue(p.DisplayName),
		TimeZone:         types.StringValue(p.TimeZone),
		CurrencyCode:     types.StringValue(p.CurrencyCode),
		IndustryCategory: types.StringValue(p.IndustryCategory),
		PropertyType:     types.StringValue(p.PropertyType),
		ServiceLevel:     types.StringValue(p.ServiceLevel),
	}
}

// knownString は plan の値が決まっているときだけその文字列を返し、null / unknown なら空文字を返す
func knownString(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}
