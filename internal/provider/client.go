package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	analyticsadmin "google.golang.org/api/analyticsadmin/v1beta"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// newAdminService は GA4 Admin API のクライアントを作る。
//
// credentials が空なら Application Default Credentials（ADC）を使う。
// ADC は `GOOGLE_APPLICATION_CREDENTIALS` のサービスアカウント JSON か、
// `gcloud auth application-default login` で作った資格情報を順に探す。
// credentials が `{` で始まればサービスアカウント JSON そのもの、それ以外はファイルパスとして扱う。
func newAdminService(ctx context.Context, credentials string) (*analyticsadmin.Service, error) {
	opts := []option.ClientOption{option.WithScopes(analyticsadmin.AnalyticsEditScope)}

	switch {
	case credentials == "":
		// ADC に任せる
	case strings.HasPrefix(strings.TrimSpace(credentials), "{"):
		opts = append(opts, option.WithCredentialsJSON([]byte(credentials)))
	default:
		if _, err := os.Stat(credentials); err != nil {
			return nil, fmt.Errorf("credentials に指定したファイルを読めません: %w", err)
		}
		opts = append(opts, option.WithCredentialsFile(credentials))
	}

	svc, err := analyticsadmin.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("GA4 Admin API クライアントの初期化に失敗しました: %w", err)
	}

	return svc, nil
}

// isNotFound は API が 404 を返したかどうかを判定する。
// Read で 404 のときは、リソースが外で消されたものとして state から外す。
func isNotFound(err error) bool {
	var gerr *googleapi.Error
	return errors.As(err, &gerr) && gerr.Code == 404
}

// serviceFromResourceConfigure は resource.Configure で渡される ProviderData から API クライアントを取り出す。
// Terraform は Configure をプロバイダの設定より前に呼ぶことがあり、そのときは ProviderData が nil で来るので、
// nil を返して呼び出し側で何もしない。
func serviceFromResourceConfigure(req resource.ConfigureRequest, diags *diag.Diagnostics) *analyticsadmin.Service {
	if req.ProviderData == nil {
		return nil
	}

	svc, ok := req.ProviderData.(*analyticsadmin.Service)
	if !ok {
		diags.AddError(
			"プロバイダの内部エラー",
			fmt.Sprintf("ProviderData の型が想定と違います: %T", req.ProviderData),
		)
		return nil
	}

	return svc
}

// serviceFromDataSourceConfigure は serviceFromResourceConfigure のデータソース版
func serviceFromDataSourceConfigure(req datasource.ConfigureRequest, diags *diag.Diagnostics) *analyticsadmin.Service {
	if req.ProviderData == nil {
		return nil
	}

	svc, ok := req.ProviderData.(*analyticsadmin.Service)
	if !ok {
		diags.AddError(
			"プロバイダの内部エラー",
			fmt.Sprintf("ProviderData の型が想定と違います: %T", req.ProviderData),
		)
		return nil
	}

	return svc
}

// parentOfName はリソース名から親のリソース名を返す。
// 例: "properties/123/dataStreams/456" → "properties/123"。
// GA4 のリソース名は「コレクション/ID」の繰り返しなので、末尾 2 要素を落とせば親になる。
func parentOfName(name string) (string, error) {
	parts := strings.Split(name, "/")
	if len(parts) < 4 || len(parts)%2 != 0 {
		return "", fmt.Errorf("リソース名の形式が想定と違います: %q", name)
	}

	return strings.Join(parts[:len(parts)-2], "/"), nil
}
