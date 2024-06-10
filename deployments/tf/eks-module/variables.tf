variable "project_name" {
  default = "prebid-server"
  type    = string
}

variable "environment" {
  type = string
}

variable "datadog_api_key" {
  type = string
}
variable "datadog_app_key" {
  type = string
}

variable "profile" {
  default = ""
}

variable "region" {
  default = "us-east-1"
  type    = string
}
