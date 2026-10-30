variable "random_bits" {
  description = "Random bits to make names and tags unique"
  type        = string
  default     = "42"
}

variable "name" {
  description = "The name of the MongoDB Tunnel Jump Item"
  type        = string
  default     = "mongodb_jump"
}

variable "hostname" {
  description = "The hostname to use."
  type        = string
  default     = "mongodb.jump.host"
}

variable "tag_suffix" {
  description = "Appended to the primary item's tag, so an update stage can change it"
  type        = string
  default     = ""
}

variable "username" {
  description = "Database username"
  type        = string
  default     = "mongouser"
}

variable "database" {
  description = "Database name"
  type        = string
  default     = "mongo"
}

variable "auth_database" {
  description = "Authentication database. Only set on PRA 26.3.1 or later."
  type        = string
  default     = ""
}

variable "tunnel_listen_address" {
  description = "Listen address (must be in 127.0.0.0/24)"
  type        = string
  default     = "127.0.0.1"
}
