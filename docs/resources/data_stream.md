# ga4_data_stream (Resource)

GA4 のウェブデータストリーム。iOS / Android のアプリストリームは Firebase が必要で API だけでは作れないため扱わない。

## 引数

- `property`（必須、変更で作り直し）: `properties/123`
- `display_name`（必須）
- `default_uri`（必須）: ストリームの URL。Measurement Protocol だけで使うなら値は動作に影響しない
- `type`（省略可、既定 `WEB_DATA_STREAM`）: `WEB_DATA_STREAM` のみ

## 属性

- `id`: `properties/123/dataStreams/456`。`ga4_measurement_protocol_secret` の `data_stream` に渡す
- `measurement_id`: `G-XXXXXXXXXX`
- `firebase_app_id`: 通常空

## Import

```bash
terraform import ga4_data_stream.x properties/123/dataStreams/456
```
