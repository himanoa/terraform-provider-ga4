variable "credentials" {
  description = "サービスアカウントの JSON キー（ファイルパスか中身）。null なら Application Default Credentials を使う"
  type        = string
  default     = null
  sensitive   = true
}

variable "account_display_name" {
  description = "既存の GA4 アカウントの表示名"
  type        = string
}

variable "property_display_name" {
  description = "作成するプロパティの表示名"
  type        = string
}

variable "event_dimensions" {
  description = "イベントスコープのカスタムディメンション。キーがパラメータ名、値が表示名"
  type        = map(string)
  default = {
    stage_id = "Stage ID"
  }
}

variable "user_dimensions" {
  description = "ユーザースコープのカスタムディメンション。キーがユーザープロパティ名、値が表示名"
  type        = map(string)
  default = {
    plan = "Plan"
  }
}
