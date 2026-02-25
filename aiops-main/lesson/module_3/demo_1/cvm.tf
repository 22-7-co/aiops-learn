module "cvm" {
  source     = "./module/cvm"
  secret_id  = var.secret_id
  secret_key = var.secret_key
  password   = var.password
  cpu        = 8
  memory     = 16
}

module "k3s" {
  depends_on  = [module.cvm]
  source      = "./module/k3s"
  public_ip   = module.cvm.public_ip
  private_ip  = module.cvm.private_ip
  server_name = "k3s-hongkong-1"
}

resource "null_resource" "connect_cvm" {
  depends_on = [module.k3s]
  connection {
    host     = module.cvm.public_ip
    type     = "ssh"
    user     = "ubuntu"
    password = var.password
  }

  triggers = {
    script_hash = filemd5("${path.module}/docker.sh")
  }

  provisioner "file" {
    source      = "docker.sh"
    destination = "/tmp/docker.sh"
  }

  provisioner "file" {
    source      = "rootfs.tar"
    destination = "/tmp/rootfs.tar"
  }

  provisioner "file" {
    source      = "./overlay"
    destination = "/tmp/overlay"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/docker.sh",
      "sh /tmp/docker.sh",
    ]
  }
}

output "cvm_public_ip" {
  value = module.cvm.public_ip
}

output "ssh_password" {
  value = var.password
}

output "kube_config" {
  description = "kubeconfig"
  value       = "export KUBECONFIG='$(pwd)/module/k3s/config.yaml'"
}
