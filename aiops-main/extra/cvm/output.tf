output "public_ip" {
  description = "vm public ip address"
  value       = module.cvm.public_ip
}

output "cvm_password" {
  description = "vm password"
  value       = var.password
}