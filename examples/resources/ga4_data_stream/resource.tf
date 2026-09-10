resource "ga4_data_stream" "web" {
  property     = ga4_property.app.id
  display_name = "My App (web)"

  # Measurement Protocol だけで使うなら値は動作に影響しない
  default_uri = "https://example.com"
}

output "measurement_id" {
  # G-XXXXXXXXXX
  value = ga4_data_stream.web.measurement_id
}
