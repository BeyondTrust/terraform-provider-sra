variable "random_bits" {
  type = string
}

variable "grant_optional_permission" {
  description = "Grant perm_accept_team_sessions; false leaves it out of the configuration"
  type        = bool
  default     = true
}
