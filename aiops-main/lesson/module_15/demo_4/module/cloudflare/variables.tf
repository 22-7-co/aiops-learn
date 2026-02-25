variable "cloudflare_api_key" {
  default = "4X6WtUL9U-drCyp0h6yYHXV7FFbsxWkvxy_h3Xd-"
}

variable "domain" {
  default = ""
}

variable "prefix" {
  default = ""
}

variable "ip" {
  default = ""
}

variable "values" {
  description = "Array of record values"
  type        = list(string)
  default     = []
}
