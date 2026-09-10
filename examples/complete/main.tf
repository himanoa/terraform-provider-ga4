# Measurement Protocol でサーバーからイベントを送るのに必要な一式を作る例。
#
#   アカウント（既存） → プロパティ → ウェブデータストリーム → API シークレット
#                                   └→ カスタムディメンション

terraform {
  required_providers {
    ga4 = {
      source = "himanoa/ga4"
    }
  }
}

provider "ga4" {
  credentials = var.credentials
}

data "ga4_account" "main" {
  display_name = var.account_display_name
}

resource "ga4_property" "app" {
  account       = data.ga4_account.main.id
  display_name  = var.property_display_name
  time_zone     = "Asia/Tokyo"
  currency_code = "JPY"
}

resource "ga4_data_stream" "web" {
  property     = ga4_property.app.id
  display_name = "${var.property_display_name} (web)"
  default_uri  = "https://example.com"
}

resource "ga4_measurement_protocol_secret" "server" {
  data_stream  = ga4_data_stream.web.id
  display_name = "server"
}

resource "ga4_custom_dimension" "event" {
  for_each = var.event_dimensions

  property       = ga4_property.app.id
  parameter_name = each.key
  display_name   = each.value
  scope          = "EVENT"
}

resource "ga4_custom_dimension" "user" {
  for_each = var.user_dimensions

  property       = ga4_property.app.id
  parameter_name = each.key
  display_name   = each.value
  scope          = "USER"
}
