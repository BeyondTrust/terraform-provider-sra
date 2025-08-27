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

variable "hostname" {
  description = "The hostname to use."
  type        = string
  default     = "net.jump.host"
}

variable "filter_rules" {
  description = "Filter rules as a list of objects. Example: [{ ip_addresses = [\"10.0.0.0/24\"] }]"
  type        = any
  default     = [
    { ip_addresses = ["10.0.0.0/24"] },
  ]
}
