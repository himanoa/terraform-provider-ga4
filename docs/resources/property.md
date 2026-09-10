# ga4_property (Resource)

GA4 プロパティ。destroy はゴミ箱への移動で、35 日間は管理画面から復元できる。

## 引数

- `account`（必須、変更で作り直し）: `accounts/123`
- `display_name`（必須）
- `time_zone`（必須）: 例 `Asia/Tokyo`
- `currency_code`（省略可）: 例 `JPY`。省略すると GA4 側の既定値
- `industry_category`（省略可）: 例 `GAMES`
- `acknowledge_user_data_collection`（省略可、既定 `true`）: 「ユーザーデータ収集の確認」を行う。
  `ga4_measurement_protocol_secret` はこれが済んでいないと作れない（API が `failedPrecondition` を返す）。
  確認済みかを返す API が無いので `false` に戻しても GA4 側は変わらない

## 属性

- `id`: `properties/123`。子リソースの `property` に渡す
- `property_type`: `PROPERTY_TYPE_ORDINARY` など
- `service_level`: `GOOGLE_ANALYTICS_STANDARD` / `GOOGLE_ANALYTICS_360`

## Import

```bash
terraform import ga4_property.x properties/123
```

## 扱えないもの

レポート用識別子・データ保持期間・Google シグナルは Admin API v1beta に無いので管理画面で設定する。
