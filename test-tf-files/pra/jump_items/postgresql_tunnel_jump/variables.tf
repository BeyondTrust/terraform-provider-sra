variable "random_bits" {
  description = "Random bits to make names and tags unique"
  type        = string
  default     = "42"
}

variable "name" {
  description = "The name of the PostgreSQL Tunnel Jump Item"
  type        = string
  default     = "pg_jump"
}

variable "hostname" {
  description = "The hostname to use."
  type        = string
  default     = "pg.jump.host"
}

variable "username" {
  description = "Database username"
  type        = string
  default     = "pguser"
}

variable "database" {
  description = "Database name"
  type        = string
  default     = "postgres"
}

variable "tunnel_listen_address" {
  description = "Optional listen address (must be in 127.0.0.0/24)"
  type        = string
  default     = "127.0.0.1"
}
