output "private_access_ip" {
  value = tencentcloud_postgresql_instance.postgresql.private_access_ip
}

output "private_access_port" {
  value = tencentcloud_postgresql_instance.postgresql.private_access_port
}

output "public_access_host" {
  value = tencentcloud_postgresql_instance.postgresql.public_access_host
}

output "public_access_port" {
  value = tencentcloud_postgresql_instance.postgresql.public_access_port
}
