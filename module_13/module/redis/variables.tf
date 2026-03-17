# Tencent Cloud variables.

variable "secret_id" {
  default = "Your Access ID"
}

variable "secret_key" {
  default = "Your Access Key"
}

variable "region" {
  description = "The location where instacne will be created"
  default     = "ap-hongkong"
}

variable "mem_size" {
  default = "1024"
}

variable "vpc_id" {
  default = ""
}

variable "subnet_id" {
  default = ""
}

# ap-hongkong-2
variable "availability_zone" {
  default = "ap-hongkong-2"
}
