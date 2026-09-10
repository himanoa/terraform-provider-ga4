# ga4_custom_dimension (Resource)

GA4 のカスタムディメンション。

**GA4 に削除は無く、destroy はアーカイブになる。** アーカイブは取り消せない（同じ `parameter_name` で
新しく作ることはできるが、アーカイブ済みのものはプロパティに残る）。`parameter_name` / `scope` を変えると「アーカイブして別名で作る」動きになる。

## 引数

- `property`（必須、変更で作り直し）: `properties/123`
- `parameter_name`（必須、変更で作り直し）: イベントパラメータ名（`EVENT`）またはユーザープロパティ名（`USER`）。
  英数字と `_`、先頭は英字、40 文字以内（`USER` は 24 文字以内）
- `display_name`（必須）: 82 文字以内
- `scope`（必須、変更で作り直し）: `EVENT` / `USER` / `ITEM`
- `description`（省略可）: 150 文字以内
- `disallow_ads_personalization`（省略可、既定 `false`）: `USER` のときだけ意味を持つ

## 属性

- `id`: `properties/123/customDimensions/456`

## Import

```bash
terraform import ga4_custom_dimension.x properties/123/customDimensions/456
```
