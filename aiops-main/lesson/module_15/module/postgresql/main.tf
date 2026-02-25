provider "tencentcloud" {
  secret_id  = var.secret_id
  secret_key = var.secret_key
  region     = var.region
}

resource "tencentcloud_postgresql_instance" "postgresql" {
  name                 = "harbor-postgresql"
  availability_zone    = var.availability_zone
  charge_type          = "POSTPAID_BY_HOUR"
  vpc_id               = var.vpc_id
  subnet_id            = var.subnet_id
  root_user            = var.username
  root_password        = var.password
  engine_version       = "15.1"
  charset              = "UTF8"
  project_id           = 0
  memory               = var.mem_size
  storage              = var.storage
  public_access_switch = true
  security_groups      = [var.security_group_id]
}
