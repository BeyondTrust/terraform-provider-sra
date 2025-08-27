variable "random_bits" {
  description = "Random bits to make names and tags unique"
  type        = string
  default     = "42"
}

variable "name" {
  description = "The name of the MySQL Tunnel Jump Item"
  type        = string
  default     = "mysql_jump"
}

variable "hostname" {
  description = "The hostname to use."
  type        = string
  default     = "mysql.jump.host"
}

variable "username" {
  description = "Database username"
  type        = string
  default     = "mysqluser"
}

variable "database" {
  description = "Database name"
  type        = string
  default     = "mysql"
}

variable "tunnel_listen_address" {
  description = "Optional listen address (must be in 127.0.0.0/24)"
  type        = string
  default     = "127.0.0.1"
}
