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

variable "availability_zone" {
  default = "ap-hongkong-2"
}

variable "mem_size" {
  default = "2"
}

variable "storage" {
  default = "50"
}

variable "vpc_id" {
  default = ""
}

variable "subnet_id" {
  default = ""
}

variable "username" {
  default = "root123"
}

variable "password" {
  default = "Root123$"
}

variable "security_group_id" {
  default = ""
}
