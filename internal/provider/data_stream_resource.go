package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	analyticsadmin "google.golang.org/api/analyticsadmin/v1beta"
)

var (
	_ resource.Resource                = (*dataStreamResource)(nil)
	_ resource.ResourceWithConfigure   = (*dataStreamResource)(nil)
	_ resource.ResourceWithImportState = (*dataStreamResource)(nil)
)

const dataStreamTypeWeb = "WEB_DATA_STREAM"

// dataStreamResource は GA4 のデータストリーム。
//
// いまはウェブストリームだけを扱う。iOS / Android のアプリストリームは Firebase アプリと紐づく必要があり、
// Admin API だけでは作れない（Measurement Protocol の送信先にもできない）。
type dataStreamResource struct {
	svc *analyticsadmin.Service
}

type dataStreamResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Property      types.String `tfsdk:"property"`
	Type          types.String `tfsdk:"type"`
	DisplayName   types.String `tfsdk:"display_name"`
	DefaultURI    types.String `tfsdk:"default_uri"`
	MeasurementID types.String `tfsdk:"measurement_id"`
	FirebaseAppID types.String `tfsdk:"firebase_app_id"`
}

func newDataStreamResource() resource.Resource {
	return &dataStreamResource{}
}

func (r *dataStreamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_stream"
}

func (r *dataStreamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "GA4 のデータストリーム。いまはウェブストリーム（`WEB_DATA_STREAM`）だけを扱う。" +
			"Measurement Protocol で送るときは `measurement_id` と `ga4_measurement_protocol_secret` の `secret_value` を使う。",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "ストリームのリソース名（`properties/123/dataStreams/456`）。`ga4_measurement_protocol_secret` の `data_stream` に渡す。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"property": schema.StringAttribute{
				Required:      true,
				Description:   "所属するプロパティのリソース名（`properties/123`）。変更すると作り直しになる。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString(dataStreamTypeWeb),
				Description:   "ストリームの種類。`WEB_DATA_STREAM` のみ。",
				Validators:    []validator.String{stringvalidator.OneOf(dataStreamTypeWeb)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"display_name": schema.StringAttribute{
				Required:    true,
				Description: "ストリームの表示名。",
			},
			"default_uri": schema.StringAttribute{
				Required:    true,
				Description: "ウェブストリームの URL（例: `https://app.example.com`）。Measurement Protocol だけで使うなら値は動作に影響しない。",
			},
			"measurement_id": schema.StringAttribute{
				Computed:      true,
				Description:   "測定 ID（`G-XXXXXXXXXX`）。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"firebase_app_id": schema.StringAttribute{
				Computed:      true,
				Description:   "Firebase と紐づいている場合のアプリ ID。ウェブストリームでは通常空。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *dataStreamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if svc := serviceFromResourceConfigure(req, &resp.Diagnostics); svc != nil {
		r.svc = svc
	}
}

func (r *dataStreamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dataStreamResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.svc.Properties.DataStreams.Create(plan.Property.ValueString(), &analyticsadmin.GoogleAnalyticsAdminV1betaDataStream{
		Type:        dataStreamTypeWeb,
		DisplayName: plan.DisplayName.ValueString(),
		WebStreamData: &analyticsadmin.GoogleAnalyticsAdminV1betaDataStreamWebStreamData{
			DefaultUri: plan.DefaultURI.ValueString(),
		},
	}).Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("GA4 データストリームを作成できません", err.Error())
		return
	}

	model, err := dataStreamModelFromAPI(created)
	if err != nil {
		resp.Diagnostics.AddError("GA4 データストリームの応答を読めません", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *dataStreamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dataStreamResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := r.svc.Properties.DataStreams.Get(state.ID.ValueString()).Context(ctx).Do()
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("GA4 データストリームを取得できません", err.Error())
		return
	}

	model, err := dataStreamModelFromAPI(got)
	if err != nil {
		resp.Diagnostics.AddError("GA4 データストリームの応答を読めません", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *dataStreamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dataStreamResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.svc.Properties.DataStreams.Patch(state.ID.ValueString(), &analyticsadmin.GoogleAnalyticsAdminV1betaDataStream{
		DisplayName: plan.DisplayName.ValueString(),
		WebStreamData: &analyticsadmin.GoogleAnalyticsAdminV1betaDataStreamWebStreamData{
			DefaultUri: plan.DefaultURI.ValueString(),
		},
	}).UpdateMask("displayName,webStreamData.defaultUri").Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("GA4 データストリームを更新できません", err.Error())
		return
	}

	model, err := dataStreamModelFromAPI(updated)
	if err != nil {
		resp.Diagnostics.AddError("GA4 データストリームの応答を読めません", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *dataStreamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dataStreamResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.svc.Properties.DataStreams.Delete(state.ID.ValueString()).Context(ctx).Do(); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("GA4 データストリームを削除できません", err.Error())
		return
	}
}

// ImportState は `terraform import ga4_data_stream.x properties/123/dataStreams/456` を受ける
func (r *dataStreamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func dataStreamModelFromAPI(s *analyticsadmin.GoogleAnalyticsAdminV1betaDataStream) (*dataStreamResourceModel, error) {
	property, err := parentOfName(s.Name)
	if err != nil {
		return nil, err
	}

	web := s.WebStreamData
	if web == nil {
		web = &analyticsadmin.GoogleAnalyticsAdminV1betaDataStreamWebStreamData{}
	}

	return &dataStreamResourceModel{
		ID:            types.StringValue(s.Name),
		Property:      types.StringValue(property),
		Type:          types.StringValue(s.Type),
		DisplayName:   types.StringValue(s.DisplayName),
		DefaultURI:    types.StringValue(web.DefaultUri),
		MeasurementID: types.StringValue(web.MeasurementId),
		FirebaseAppID: types.StringValue(web.FirebaseAppId),
	}, nil
}
