terraform {
  required_version = "> 0.13.0"
  required_providers {
    tencentcloud = {
      source  = "tencentcloudstack/tencentcloud"
      version = "1.81.5"
    }
  }
}

provider "tencentcloud" {
  secret_id  = var.secret_id
  secret_key = var.secret_key
  region     = var.region
}

resource "tencentcloud_image" "image_snap" {
  image_name        = "ollama-gpu-image"
  force_poweroff    = true
  image_description = "ollama gpu image"
  instance_id       = module.cvm.instance_id
  depends_on        = [null_resource.connect_cvm]
}
