provider "tencentcloud" {
  secret_id  = var.secret_id
  secret_key = var.secret_key
  region     = var.region
}

data "tencentcloud_redis_zone_config" "zone" {
  type_id = 15
}

# resource "tencentcloud_vpc" "vpc" {
#   cidr_block = "10.0.0.0/16"
#   name       = "tf_redis_vpc"
# }

# resource "tencentcloud_subnet" "subnet" {
#   vpc_id            = tencentcloud_vpc.vpc.id
#   availability_zone = data.tencentcloud_redis_zone_config.zone.list[0].zone
#   name              = "tf_redis_subnet"
#   cidr_block        = "10.0.1.0/24"
# }

resource "tencentcloud_redis_instance" "redis" {
  availability_zone = var.availability_zone
  type_id           = data.tencentcloud_redis_zone_config.zone.list[0].type_id
  no_auth           = true
  mem_size          = 1024
  name              = "harbor-redis"
  port              = 6379
  vpc_id            = var.vpc_id
  subnet_id         = var.subnet_id
}
