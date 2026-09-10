package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	analyticsadmin "google.golang.org/api/analyticsadmin/v1beta"
)

var (
	_ resource.Resource                = (*customDimensionResource)(nil)
	_ resource.ResourceWithConfigure   = (*customDimensionResource)(nil)
	_ resource.ResourceWithImportState = (*customDimensionResource)(nil)
)

// customDimensionResource は GA4 のカスタムディメンション。
//
// GA4 にはカスタムディメンションの削除が無く、アーカイブしかない。アーカイブは取り消せない
// （同じパラメータ名で新しく作ることはできる）。そのため destroy はアーカイブとして実装している。
// `parameter_name` や `scope` を変えると「アーカイブして別名で作る」動きになる点に注意。
type customDimensionResource struct {
	svc *analyticsadmin.Service
}

type customDimensionResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	Property                   types.String `tfsdk:"property"`
	ParameterName              types.String `tfsdk:"parameter_name"`
	DisplayName                types.String `tfsdk:"display_name"`
	Description                types.String `tfsdk:"description"`
	Scope                      types.String `tfsdk:"scope"`
	DisallowAdsPersonalization types.Bool   `tfsdk:"disallow_ads_personalization"`
}

func newCustomDimensionResource() resource.Resource {
	return &customDimensionResource{}
}

func (r *customDimensionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_dimension"
}

func (r *customDimensionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "GA4 のカスタムディメンション。GA4 に削除は無いので、destroy はアーカイブになる。" +
			"アーカイブは取り消せない（同じ `parameter_name` で新しく作ることはできる）。" +
			"`parameter_name` / `scope` の変更は「アーカイブして別名で作り直す」動きになる。",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "カスタムディメンションのリソース名（`properties/123/customDimensions/456`）。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"property": schema.StringAttribute{
				Required:      true,
				Description:   "所属するプロパティのリソース名（`properties/123`）。変更すると作り直しになる。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"parameter_name": schema.StringAttribute{
				Required: true,
				Description: "イベントパラメータ名（`scope` が `EVENT` のとき）またはユーザープロパティ名（`USER` のとき）。" +
					"英数字と `_` のみ、先頭は英字、40 文字以内（USER は 24 文字以内）。変更すると作り直し（＝旧いものはアーカイブ）になる。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"display_name": schema.StringAttribute{
				Required:    true,
				Description: "レポートに出る表示名。82 文字以内。",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "説明。150 文字以内。",
			},
			"scope": schema.StringAttribute{
				Required:      true,
				Description:   "`EVENT`、`USER`、`ITEM` のいずれか。変更すると作り直しになる。",
				Validators:    []validator.String{stringvalidator.OneOf("EVENT", "USER", "ITEM")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"disallow_ads_personalization": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "広告のパーソナライズに使わない（NPA）指定。`scope` が `USER` のときだけ意味を持つ。",
			},
		},
	}
}

func (r *customDimensionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if svc := serviceFromResourceConfigure(req, &resp.Diagnostics); svc != nil {
		r.svc = svc
	}
}

func (r *customDimensionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customDimensionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.svc.Properties.CustomDimensions.Create(plan.Property.ValueString(), &analyticsadmin.GoogleAnalyticsAdminV1betaCustomDimension{
		ParameterName:              plan.ParameterName.ValueString(),
		DisplayName:                plan.DisplayName.ValueString(),
		Description:                plan.Description.ValueString(),
		Scope:                      plan.Scope.ValueString(),
		DisallowAdsPersonalization: plan.DisallowAdsPersonalization.ValueBool(),
	}).Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("GA4 カスタムディメンションを作成できません", err.Error())
		return
	}

	model, err := customDimensionModelFromAPI(created)
	if err != nil {
		resp.Diagnostics.AddError("GA4 カスタムディメンションの応答を読めません", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *customDimensionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customDimensionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := r.svc.Properties.CustomDimensions.Get(state.ID.ValueString()).Context(ctx).Do()
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GA4 カスタムディメンションを取得できません", err.Error())
		return
	}

	model, err := customDimensionModelFromAPI(got)
	if err != nil {
		resp.Diagnostics.AddError("GA4 カスタムディメンションの応答を読めません", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *customDimensionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state customDimensionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// description を空に戻すときも送りたいので ForceSendFields で空文字を落とさせない
	updated, err := r.svc.Properties.CustomDimensions.Patch(state.ID.ValueString(), &analyticsadmin.GoogleAnalyticsAdminV1betaCustomDimension{
		DisplayName:                plan.DisplayName.ValueString(),
		Description:                plan.Description.ValueString(),
		DisallowAdsPersonalization: plan.DisallowAdsPersonalization.ValueBool(),
		ForceSendFields:            []string{"Description", "DisallowAdsPersonalization"},
	}).UpdateMask("displayName,description,disallowAdsPersonalization").Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("GA4 カスタムディメンションを更新できません", err.Error())
		return
	}

	model, err := customDimensionModelFromAPI(updated)
	if err != nil {
		resp.Diagnostics.AddError("GA4 カスタムディメンションの応答を読めません", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Delete はアーカイブする。GA4 にカスタムディメンションの削除は無い
func (r *customDimensionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customDimensionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.svc.Properties.CustomDimensions.Archive(
		state.ID.ValueString(),
		&analyticsadmin.GoogleAnalyticsAdminV1betaArchiveCustomDimensionRequest{},
	).Context(ctx).Do()
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("GA4 カスタムディメンションをアーカイブできません", err.Error())
		return
	}
}

// ImportState は `terraform import ga4_custom_dimension.x properties/123/customDimensions/456` を受ける
func (r *customDimensionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func customDimensionModelFromAPI(d *analyticsadmin.GoogleAnalyticsAdminV1betaCustomDimension) (*customDimensionResourceModel, error) {
	property, err := parentOfName(d.Name)
	if err != nil {
		return nil, err
	}

	return &customDimensionResourceModel{
		ID:                         types.StringValue(d.Name),
		Property:                   types.StringValue(property),
		ParameterName:              types.StringValue(d.ParameterName),
		DisplayName:                types.StringValue(d.DisplayName),
		Description:                types.StringValue(d.Description),
		Scope:                      types.StringValue(d.Scope),
		DisallowAdsPersonalization: types.BoolValue(d.DisallowAdsPersonalization),
	}, nil
}
