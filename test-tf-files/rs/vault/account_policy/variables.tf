variable "random_bits" {
  description = "Random bits to make names and tags unique"
  type        = string
  default     = "42"
}

variable "name" {
  description = "The name of the Vault Account Policy"
  type        = string
  default     = "fun_policy"
}
