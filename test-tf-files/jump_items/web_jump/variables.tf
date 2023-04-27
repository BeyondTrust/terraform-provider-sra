variable "random_bits" {
  description = "Random bits to make names and tags unique"
  type        = string
  default     = "42"
}

variable "name" {
  description = "The name of the Web Jump Item"
  type        = string
  default     = "fun_jump"
}

variable "url" {
  description = "The url to use."
  type        = string
  default     = "https://fun.jump.host"
}
