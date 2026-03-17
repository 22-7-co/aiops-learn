resource "null_resource" "connect_ubuntu" {
  depends_on = [module.k3s]
  connection {
    host     = tencentcloud_instance.web[0].public_ip
    type     = "ssh"
    user     = "ubuntu"
    password = var.password
  }

  triggers = {
    script_hash = filemd5("${path.module}/init.sh")
  }

  provisioner "file" {
    destination = "/tmp/init.sh"
    source      = "${path.module}/init.sh"
  }

  provisioner "file" {
    destination = "/tmp/values.yaml"
    source      = "${path.module}/values.yaml"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/init.sh",
      "sh /tmp/init.sh",
    ]
  }
}
