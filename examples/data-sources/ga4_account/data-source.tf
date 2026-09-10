# 表示名でアカウントを引く。完全一致で 1 件に絞れる必要がある
data "ga4_account" "main" {
  display_name = "My Company"
}

output "account_id" {
  # accounts/123 の形式。ga4_property の account に渡す
  value = data.ga4_account.main.id
}
