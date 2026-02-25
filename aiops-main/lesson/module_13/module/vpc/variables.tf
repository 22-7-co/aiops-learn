variable "secret_id" {
  default = "Your Access ID"
}

variable "secret_key" {
  default = "Your Access Key"
}

variable "region" {
  default = "ap-hongkong"
}

# default zone is ap-hongkong-2
# https://www.tencentcloud.com/document/product/239/4106?from_cn_redirect=1
variable "availability_zone" {
  default = "ap-hongkong-2"
}
