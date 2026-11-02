variable "random_bits" {
  type = string
}

variable "optional_permission" {
  description = "perm_invite_external_user; null leaves it out of the configuration"
  type        = bool
  default     = true
}
