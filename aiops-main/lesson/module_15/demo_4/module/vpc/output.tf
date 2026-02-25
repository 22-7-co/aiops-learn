output "vpc_id" {
  value = tencentcloud_vpc.vpc.id
}

output "subnet_id" {
  value = tencentcloud_subnet.subnet.id
}

output "security_group_id" {
  value = tencentcloud_security_group.default.id
}
