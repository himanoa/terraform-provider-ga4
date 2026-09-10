# ga4_measurement_protocol_secret (Resource)

Measurement Protocol の API シークレット。1 ストリームにつき 10 個まで。

## 引数

- `data_stream`（必須、変更で作り直し）: `properties/123/dataStreams/456`
- `display_name`（必須）: 管理画面の「ニックネーム」

## 属性

- `id`: `properties/123/dataStreams/456/measurementProtocolSecrets/789`
- `secret_value`（秘匿）: Measurement Protocol の `api_secret` に渡す値。**state に平文で入る**

## Import

```bash
terraform import ga4_measurement_protocol_secret.x properties/123/dataStreams/456/measurementProtocolSecrets/789
```
