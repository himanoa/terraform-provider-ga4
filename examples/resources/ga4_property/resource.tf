data "ga4_account" "main" {
  display_name = "My Company"
}

resource "ga4_property" "app" {
  account      = data.ga4_account.main.id
  display_name = "My App"
  time_zone    = "Asia/Tokyo"

  # 省略可
  currency_code     = "JPY"
  industry_category = "GAMES"
}
