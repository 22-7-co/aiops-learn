module "cvm" {
  source     = "./module/cvm"
  secret_id  = var.secret_id
  secret_key = var.secret_key
  password   = var.password
  cpu        = 8
  memory     = 32
}

resource "null_resource" "connect_cvm" {
  connection {
    host     = module.cvm.public_ip
    type     = "ssh"
    user     = "ubuntu"
    password = var.password
  }

  triggers = {
    script_hash = filemd5("${path.module}/llm.sh")
  }

  provisioner "file" {
    source      = "llm.sh"
    destination = "/tmp/llm.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/llm.sh",
      "sh /tmp/llm.sh",
    ]
  }
}

output "cvm_public_ip" {
  value = module.cvm.public_ip
}

output "ssh_password" {
  value = var.password
}

output "image_id" {
  value = tencentcloud_image.image_snap.id
}

output "vpc_id" {
  value = module.cvm.vpc_id
}

output "subnet_id" {
  value = module.cvm.subnet_id
}

output "security_group_id" {
  value = module.cvm.security_group_id
}

output "region" {
  value = var.region
}

output "availability_zone" {
  value = module.cvm.availability_zone
}
