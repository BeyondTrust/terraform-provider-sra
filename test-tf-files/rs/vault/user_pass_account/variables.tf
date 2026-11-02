variable "random_bits" {
  description = "Random bits to make names and tags unique"
  type        = string
  default     = "42"
}

variable "name" {
  description = "The name of the Vault Account"
  type        = string
  default     = "fun_account"
}

# ephemeral keeps the value out of saved plans, which record every other input
# variable's value.
variable "write_only_password" {
  description = "The password for the write_only account"
  type        = string
  sensitive   = true
  ephemeral   = true
  default     = "write-only-password"
}

variable "password_wo_version" {
  description = "Changing this sends write_only_password again"
  type        = number
  default     = 1
}
