variable "random_bits" {
  description = "Random bits to make names and tags unique"
  type        = string
  default     = "42"
}

variable "name" {
  description = "The name of the VNC Jump Item"
  type        = string
  default     = "fun_jump"
}

variable "hostname" {
  description = "The hostname to use."
  type        = string
  default     = "fun.jump.host"
}
