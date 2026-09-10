package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	analyticsadmin "google.golang.org/api/analyticsadmin/v1beta"
)

var (
	_ resource.Resource                = (*measurementProtocolSecretResource)(nil)
	_ resource.ResourceWithConfigure   = (*measurementProtocolSecretResource)(nil)
	_ resource.ResourceWithImportState = (*measurementProtocolSecretResource)(nil)
)

// measurementProtocolSecretResource は Measurement Protocol の API シークレット。
// `secret_value` は state に平文で入るので、state の置き場所には気をつけること。
type measurementProtocolSecretResource struct {
	svc *analyticsadmin.Service
}

type measurementProtocolSecretResourceModel struct {
	ID          types.String `tfsdk:"id"`
	DataStream  types.String `tfsdk:"data_stream"`
	DisplayName types.String `tfsdk:"display_name"`
	SecretValue types.String `tfsdk:"secret_value"`
}

func newMeasurementProtocolSecretResource() resource.Resource {
	return &measurementProtocolSecretResource{}
}

func (r *measurementProtocolSecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_measurement_protocol_secret"
}

func (r *measurementProtocolSecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Measurement Protocol の API シークレット。1 ストリームにつき 10 個まで。" +
			"`secret_value` は state に平文で保存される。",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "シークレットのリソース名（`properties/123/dataStreams/456/measurementProtocolSecrets/789`）。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"data_stream": schema.StringAttribute{
				Required:      true,
				Description:   "所属するデータストリームのリソース名。`ga4_data_stream` の `id` を渡す。変更すると作り直しになる。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"display_name": schema.StringAttribute{
				Required:    true,
				Description: "シークレットの表示名（管理画面の「ニックネーム」）。",
			},
			"secret_value": schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "Measurement Protocol の `api_secret` に渡す値。",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *measurementProtocolSecretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if svc := serviceFromResourceConfigure(req, &resp.Diagnostics); svc != nil {
		r.svc = svc
	}
}

func (r *measurementProtocolSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan measurementProtocolSecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.svc.Properties.DataStreams.MeasurementProtocolSecrets.Create(
		plan.DataStream.ValueString(),
		&analyticsadmin.GoogleAnalyticsAdminV1betaMeasurementProtocolSecret{DisplayName: plan.DisplayName.ValueString()},
	).Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("Measurement Protocol API シークレットを作成できません", err.Error())
		return
	}

	model, err := measurementProtocolSecretModelFromAPI(created)
	if err != nil {
		resp.Diagnostics.AddError("Measurement Protocol API シークレットの応答を読めません", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *measurementProtocolSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state measurementProtocolSecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := r.svc.Properties.DataStreams.MeasurementProtocolSecrets.Get(state.ID.ValueString()).Context(ctx).Do()
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Measurement Protocol API シークレットを取得できません", err.Error())
		return
	}

	model, err := measurementProtocolSecretModelFromAPI(got)
	if err != nil {
		resp.Diagnostics.AddError("Measurement Protocol API シークレットの応答を読めません", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *measurementProtocolSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state measurementProtocolSecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.svc.Properties.DataStreams.MeasurementProtocolSecrets.Patch(
		state.ID.ValueString(),
		&analyticsadmin.GoogleAnalyticsAdminV1betaMeasurementProtocolSecret{DisplayName: plan.DisplayName.ValueString()},
	).UpdateMask("displayName").Context(ctx).Do()
	if err != nil {
		resp.Diagnostics.AddError("Measurement Protocol API シークレットを更新できません", err.Error())
		return
	}

	model, err := measurementProtocolSecretModelFromAPI(updated)
	if err != nil {
		resp.Diagnostics.AddError("Measurement Protocol API シークレットの応答を読めません", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *measurementProtocolSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state measurementProtocolSecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.svc.Properties.DataStreams.MeasurementProtocolSecrets.Delete(state.ID.ValueString()).Context(ctx).Do(); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Measurement Protocol API シークレットを削除できません", err.Error())
		return
	}
}

// ImportState は `terraform import ga4_measurement_protocol_secret.x properties/123/dataStreams/456/measurementProtocolSecrets/789` を受ける
func (r *measurementProtocolSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func measurementProtocolSecretModelFromAPI(s *analyticsadmin.GoogleAnalyticsAdminV1betaMeasurementProtocolSecret) (*measurementProtocolSecretResourceModel, error) {
	dataStream, err := parentOfName(s.Name)
	if err != nil {
		return nil, err
	}

	return &measurementProtocolSecretResourceModel{
		ID:          types.StringValue(s.Name),
		DataStream:  types.StringValue(dataStream),
		DisplayName: types.StringValue(s.DisplayName),
		SecretValue: types.StringValue(s.SecretValue),
	}, nil
}
