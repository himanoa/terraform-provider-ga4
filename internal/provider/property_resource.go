package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	// GA4 側から読み戻せない（確認済みかを返す API が無い）ので、Read では直前の state の値を引き継ぐ
	AcknowledgeUserDataCollection types.Bool `tfsdk:"acknowledge_user_data_collection"`
}

// userDataCollectionAcknowledgement は API が要求する定型文。1 文字でも違うと 400 になる
const userDataCollectionAcknowledgement = "I acknowledge that I have the necessary privacy disclosures and rights from my end users " +
	"for the collection and processing of their data, including the association of such data with the visitation " +
	"information Google Analytics collects from my site and/or app property."

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
			"acknowledge_user_data_collection": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "「ユーザーデータ収集の確認」をこのプロパティに対して行う（管理画面で新規プロパティに出る確認と同じもの）。" +
					"Measurement Protocol API シークレットは、これが済んでいないと作れない。" +
					"確認済みかどうかを返す API が無いので、true → false に戻しても GA4 側は変わらない。",
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

	model := propertyModelFromAPI(created)
	model.AcknowledgeUserDataCollection = plan.AcknowledgeUserDataCollection

	// 作成に成功した時点で state に入れておく。以降で失敗しても、作ったプロパティが野良にならないようにする
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AcknowledgeUserDataCollection.ValueBool() {
		if err := r.acknowledgeUserDataCollection(ctx, created.Name); err != nil {
			resp.Diagnostics.AddError("GA4 プロパティのユーザーデータ収集の確認ができません", err.Error())
			return
		}
	}
}

// acknowledgeUserDataCollection は「ユーザーデータ収集の確認」を行う。何度呼んでも同じ結果になる
func (r *propertyResource) acknowledgeUserDataCollection(ctx context.Context, name string) error {
	_, err := r.svc.Properties.AcknowledgeUserDataCollection(name, &analyticsadmin.GoogleAnalyticsAdminV1betaAcknowledgeUserDataCollectionRequest{
		Acknowledgement: userDataCollectionAcknowledgement,
	}).Context(ctx).Do()

	return err
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

	model := propertyModelFromAPI(got)
	// API から読み戻せないので state の値を引き継ぐ（import 直後は null で、次の plan で既定値へ寄る）
	model.AcknowledgeUserDataCollection = state.AcknowledgeUserDataCollection

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
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

	// false（または import 直後の null）から true に変わったときだけ確認を行う。true → false は GA4 側で戻せないので何もしない
	if plan.AcknowledgeUserDataCollection.ValueBool() && !state.AcknowledgeUserDataCollection.ValueBool() {
		if err := r.acknowledgeUserDataCollection(ctx, state.ID.ValueString()); err != nil {
			resp.Diagnostics.AddError("GA4 プロパティのユーザーデータ収集の確認ができません", err.Error())
			return
		}
	}

	model := propertyModelFromAPI(updated)
	model.AcknowledgeUserDataCollection = plan.AcknowledgeUserDataCollection

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
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
		// 呼び出し側で plan / state の値に差し替える
		AcknowledgeUserDataCollection: types.BoolNull(),
	}
}

// knownString は plan の値が決まっているときだけその文字列を返し、null / unknown なら空文字を返す
func knownString(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}
