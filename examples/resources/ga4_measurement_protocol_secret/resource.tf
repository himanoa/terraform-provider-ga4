resource "ga4_measurement_protocol_secret" "server" {
  data_stream  = ga4_data_stream.web.id
  display_name = "server"
}

output "api_secret" {
  # Measurement Protocol の api_secret に渡す値。state には平文で入る
  value     = ga4_measurement_protocol_secret.server.secret_value
  sensitive = true
}
