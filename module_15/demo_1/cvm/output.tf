output "public_ip" {
  description = "vm public ip address"
  value       = module.cvm.public_ip
}

output "kube_config" {
  description = "kubeconfig"
  value       = "${path.module}/config.yaml"
}

output "cvm_password" {
  description = "vm password"
  value       = var.password
}

output "grafana" {
  description = "grafana url"
  value       = "${module.cvm.public_ip}:30080, admin/password123"
}

output "prometheus" {
  description = "prometheus url"
  value       = "${module.cvm.public_ip}:30090"
}
