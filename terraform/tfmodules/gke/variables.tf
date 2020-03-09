variable "enabled" {
  default = 0
}

variable "region" {
  type = "string"
}

variable "primary_zone" {
  type = "string"
}

variable "gcp_projectid" {
  type = "string"
}

variable "gcp_projectnumber" {
  type = "string"
}

variable "autoscaling_min" {
  type = "string"
}

variable "autoscaling_max" {
  type = "string"
}

variable "vm_type" {
  type = "string"
}