variable "random_bits" {
  type = string
}

variable "optional_permission" {
  description = "perm_accept_team_sessions; null leaves it out of the configuration"
  type        = bool
  default     = true
}
