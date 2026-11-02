variable "random_bits" {
  description = "Random bits to make names and tags unique"
  type        = string
  default     = "42"
}

variable "name" {
  description = "The name of the Vault Account Group"
  type        = string
  default     = "fun_account"
}

variable "extra_tags" {
  description = "Criteria tags added to new_account_group's jump item association after the first apply"
  type        = list(string)
  default     = []
}
