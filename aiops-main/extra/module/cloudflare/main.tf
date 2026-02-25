provider "cloudflare" {
  api_token = var.cloudflare_api_key
}

data "cloudflare_zone" "this" {
  name = var.domain
}

resource "cloudflare_record" "harbor" {
  for_each = toset(var.values)

  zone_id         = data.cloudflare_zone.this.id
  name            = "${each.value}.${var.prefix}.${var.domain}"
  value           = var.ip
  type            = "A"
  ttl             = 60
  allow_overwrite = true
}
