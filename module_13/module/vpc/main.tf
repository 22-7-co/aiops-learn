provider "tencentcloud" {
  secret_id  = var.secret_id
  secret_key = var.secret_key
  region     = var.region
}

resource "tencentcloud_vpc" "vpc" {
  cidr_block = "10.0.0.0/16"
  name       = "tf_default_vpc"
}

resource "tencentcloud_subnet" "subnet" {
  vpc_id            = tencentcloud_vpc.vpc.id
  availability_zone = var.availability_zone
  name              = "tf_default_subnet"
  cidr_block        = "10.0.1.0/24"
}

resource "tencentcloud_security_group" "default" {
  name = "tf-default-sg"
}

# 放通所有端口，在生产环境下请谨慎使用
resource "tencentcloud_security_group_lite_rule" "all" {
  security_group_id = tencentcloud_security_group.default.id

  ingress = [
    "ACCEPT#0.0.0.0/0#ALL#TCP",
  ]

  egress = [
    "ACCEPT#0.0.0.0/0#ALL#TCP",
  ]
}
