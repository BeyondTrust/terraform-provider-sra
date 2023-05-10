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

variable "private_key" {
  description = "The private key."
  type        = string
  default     = "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtz\nc2gtZWQyNTUxOQAAACAvEITV/TvFiDeV5hfkMqsLXOZJE3XYD4e+xZgxHq7NXgAA\nAIjpz6L86c+i/AAAAAtzc2gtZWQyNTUxOQAAACAvEITV/TvFiDeV5hfkMqsLXOZJ\nE3XYD4e+xZgxHq7NXgAAAEAXMM7hvnjZJbl3zyoeK7nvru00hCJOsT8M14eDiNo+\nCC8QhNX9O8WIN5XmF+Qyqwtc5kkTddgPh77FmDEers1eAAAAAAECAwQF\n-----END OPENSSH PRIVATE KEY-----\n"
}
