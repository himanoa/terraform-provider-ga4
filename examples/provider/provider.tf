terraform {
  required_providers {
    ga4 = {
      source = "himanoa/ga4"
    }
  }
}

# credentials を省略すると Application Default Credentials を使う
# （GOOGLE_APPLICATION_CREDENTIALS または gcloud auth application-default login）
provider "ga4" {}

# サービスアカウントの JSON キーを明示する場合。ファイルパスでも JSON の中身でもよい
# provider "ga4" {
#   credentials = file("service-account.json")
# }
