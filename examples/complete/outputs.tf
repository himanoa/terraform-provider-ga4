output "property_id" {
  value = ga4_property.app.id
}

output "measurement_id" {
  description = "Measurement Protocol の measurement_id に渡す値"
  value       = ga4_data_stream.web.measurement_id
}

output "api_secret" {
  description = "Measurement Protocol の api_secret に渡す値"
  value       = ga4_measurement_protocol_secret.server.secret_value
  sensitive   = true
}
