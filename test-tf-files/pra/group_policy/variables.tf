variable "random_bits" {
  type = string
}

variable "grant_optional_permission" {
  description = "Grant perm_invite_external_user; false leaves it out of the configuration"
  type        = bool
  default     = true
}
