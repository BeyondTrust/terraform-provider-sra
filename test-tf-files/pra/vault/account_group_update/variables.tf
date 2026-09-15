variable "random_bits" {
  description = "Random bits to make names and tags unique"
  type        = string
  default     = "42"
}

variable "name" {
  description = "The name of the Vault Account Group"
  type        = string
  default     = "This is a Name"
}

variable "with_gp_membership" {
  description = "Whether the account group declares a group policy membership"
  type        = bool
  default     = true
}
