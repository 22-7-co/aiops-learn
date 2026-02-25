module "cvm" {
  source     = "./module/cvm"
  secret_id  = var.secret_id
  secret_key = var.secret_key
  password   = var.password
  cpu        = 4
  memory     = 8
}

resource "null_resource" "connect_cvm" {
  connection {
    host     = module.cvm.public_ip
    type     = "ssh"
    user     = "ubuntu"
    password = var.password
  }

  triggers = {
    script_hash = filemd5("${path.module}/docker.sh")
    script_hash = filemd5("${path.module}/docker-compose/docker-compose.yaml")
  }

  provisioner "file" {
    source      = "docker.sh"
    destination = "/tmp/docker.sh"
  }

  provisioner "file" {
    source      = "docker-compose"
    destination = "/home/ubuntu/"
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

output "kong_server" {
  value = "${module.cvm.public_ip}:8000"
}

output "kong_api" {
  value = "${module.cvm.public_ip}:8001"
}

output "kong_webui" {
  value = "${module.cvm.public_ip}:8002"
}
