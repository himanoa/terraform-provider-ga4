# イベントスコープ: イベントパラメータをディメンションとして使う
resource "ga4_custom_dimension" "stage_id" {
  property       = ga4_property.app.id
  parameter_name = "stage_id"
  display_name   = "Stage ID"
  scope          = "EVENT"
  description    = "プレイ中のステージ"
}

# ユーザースコープ: ユーザープロパティをディメンションとして使う
resource "ga4_custom_dimension" "plan" {
  property       = ga4_property.app.id
  parameter_name = "plan"
  display_name   = "Plan"
  scope          = "USER"

  # 広告のパーソナライズに使わない（USER スコープのみ意味を持つ）
  disallow_ads_personalization = true
}
