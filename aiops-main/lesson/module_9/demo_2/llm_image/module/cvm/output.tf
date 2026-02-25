output "public_ip" {
  description = "The public ip of instance."
  value       = tencentcloud_instance.ubuntu.public_ip
}

output "private_ip" {
  description = "The private ip of instance."
  value       = tencentcloud_instance.ubuntu.private_ip
}

output "ssh_password" {
  description = "The SSH password of instance."
  value       = var.password
}

output "instance_id" {
  description = "The instance id of instance."
  value       = tencentcloud_instance.ubuntu.id
}

output "vpc_id" {
  description = "The vpc id of instance."
  value       = tencentcloud_vpc.tf_vpc.id
}

output "subnet_id" {
  description = "The subnet id of instance."
  value       = tencentcloud_subnet.tf_service_subnet.id
}

output "security_group_id" {
  description = "The security group id of instance."
  value       = tencentcloud_security_group.web_sg.id
}

output "availability_zone" {
  value = var.availability_zone
}
