output "public_ip" {
  description = "vm public ip address"
  value       = tencentcloud_instance.web[0].public_ip
}

output "kube_config" {
  description = "kubeconfig"
  value       = "${path.module}/config.yaml"
}

output "vm_password" {
  description = "vm password"
  value       = var.password
}

output "grafana_username" {
  description = "grafana username"
  value       = "admin"
}

output "grafana_password" {
  description = "grafana password"
  value       = "loki123"
}

output "grafana_url" {
  description = "grafana url"
  value       = "http://${tencentcloud_instance.web[0].public_ip}:31001"
}
