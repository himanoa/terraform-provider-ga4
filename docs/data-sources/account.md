# ga4_account (Data Source)

表示名で GA4 アカウントを引く。アカウントは API で作れないので、データソースだけ用意している。

## 引数

- `display_name`（必須）: アカウントの表示名。完全一致で 1 件に絞れる必要がある

## 属性

- `id`: アカウントのリソース名（`accounts/123`）。`ga4_property` の `account` に渡す
- `region_code`: 国コード（例: `JP`）
