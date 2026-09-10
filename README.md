# terraform-provider-ga4

Google Analytics 4 の [Admin API](https://developers.google.com/analytics/devguides/config/admin/v1) を Terraform から扱うプロバイダ。

Measurement Protocol でサーバーからイベントを送るのに必要な最小限のものだけを扱う。

| 種別 | 名前 | 内容 |
| --- | --- | --- |
| Data Source | `ga4_account` | 表示名で既存アカウントを引く（アカウントは API で作れない） |
| Resource | `ga4_property` | プロパティ |
| Resource | `ga4_data_stream` | ウェブデータストリーム |
| Resource | `ga4_measurement_protocol_secret` | Measurement Protocol の API シークレット |
| Resource | `ga4_custom_dimension` | カスタムディメンション |

詳細は [docs/](docs/) と [examples/](examples/) を見ること。

## 使い方

```hcl
terraform {
  required_providers {
    ga4 = {
      source = "himanoa/ga4"
    }
  }
}

provider "ga4" {}

data "ga4_account" "main" {
  display_name = "My Company"
}

resource "ga4_property" "app" {
  account      = data.ga4_account.main.id
  display_name = "My App"
  time_zone    = "Asia/Tokyo"
}

resource "ga4_data_stream" "web" {
  property     = ga4_property.app.id
  display_name = "My App (web)"
  default_uri  = "https://example.com"
}

resource "ga4_measurement_protocol_secret" "server" {
  data_stream  = ga4_data_stream.web.id
  display_name = "server"
}

resource "ga4_custom_dimension" "stage_id" {
  property       = ga4_property.app.id
  parameter_name = "stage_id"
  display_name   = "Stage ID"
  scope          = "EVENT"
}
```

`ga4_data_stream.web.measurement_id` と `ga4_measurement_protocol_secret.server.secret_value` を
Measurement Protocol の `measurement_id` / `api_secret` に渡す。

一式をまとめた例は [examples/complete](examples/complete/) にある。

## 認証

`credentials` にサービスアカウントの JSON キー（ファイルパスか中身）を渡す。
省略すると Application Default Credentials を使う。

```sh
# 環境変数で渡す
export GOOGLE_APPLICATION_CREDENTIALS=service-account.json

# または自分のユーザーで
gcloud auth application-default login
```

いずれの場合も、その資格情報の主体（サービスアカウントのメールアドレス、またはユーザー）を
GA4 の管理画面でアカウントかプロパティに「編集者」として追加しておくこと。
Google Cloud 側では Google Analytics Admin API を有効にしておく。

## 注意

- `ga4_property` の destroy はゴミ箱への移動。35 日間は管理画面から復元できる
- `ga4_custom_dimension` の destroy はアーカイブ。取り消せない（同じ `parameter_name` で新しく作ることはできる）
- `ga4_measurement_protocol_secret` の `secret_value` は state に平文で入る
- iOS / Android のアプリストリームは Firebase が必要で API だけでは作れないため扱わない
- レポート用識別子・データ保持期間・Google シグナルは Admin API v1beta に無いので管理画面で設定する

## 開発

```sh
make build    # ./terraform-provider-ga4 をビルド
make test     # ユニットテスト
make testacc  # 実 API を叩くテスト。GA4 の編集権限を持つ資格情報が要る
make fmt
make vet
```

ローカルのビルドを Terraform から使うには `~/.terraformrc` に dev_overrides を書く。

```hcl
provider_installation {
  dev_overrides {
    "himanoa/ga4" = "/Users/you/go/bin"
  }
  direct {}
}
```

`make install` で `$GOBIN`（既定 `~/go/bin`）に入る。dev_overrides を使うときは `terraform init` は不要。

## リリース

`v*` タグを push すると GitHub Actions が goreleaser でビルドし、GitHub Releases に置く。
署名用に `GPG_PRIVATE_KEY` と `PASSPHRASE` のシークレットが要る。

## License

[MIT](LICENSE)
