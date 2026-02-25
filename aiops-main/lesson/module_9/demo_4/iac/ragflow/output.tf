output "public_ip" {
  description = "vm public ip address"
  value       = tencentcloud_instance.web[0].public_ip
}

output "vm_password" {
  description = "vm password"
  value       = var.password
}

output "ragflow_ip" {
  description = "ragflow ip"
  value       = "http://${tencentcloud_instance.web[0].public_ip}"
}
