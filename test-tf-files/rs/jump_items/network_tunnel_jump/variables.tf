variable "random_bits" {
  description = "Random bits to make names and tags unique"
  type        = string
  default     = "42"
}

variable "name" {
  description = "The name of the Network Tunnel Jump Item"
  type        = string
  default     = "net_jump"
}
