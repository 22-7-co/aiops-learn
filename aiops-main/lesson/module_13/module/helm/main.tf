provider "helm" {
  kubernetes {
    config_path = var.filename
  }
}

resource "helm_release" "helm_charts" {
  for_each = { for chart in var.helm_charts : chart.name => chart }

  name             = each.value.name
  repository       = each.value.repository
  chart            = each.value.chart
  namespace        = each.value.namespace
  create_namespace = each.value.create_namespace
  version          = each.value.version
  values           = each.value.values_file != "" ? ["${file("${each.value.values_file}")}"] : []
  upgrade = {
    enable  = true
    install = true
  }

  dynamic "set" {
    for_each = each.value.set
    content {
      name  = set.value["name"]
      value = set.value["value"]
    }
  }
}

# example
# resource "helm_release" "example" {
#   name       = "my-redis-release"
#   repository = "https://charts.bitnami.com/bitnami"
#   chart      = "redis"
#   version    = "6.0.1"

#   values = [
#     "${file("values.yaml")}"
#   ]

#   set {
#     name  = "cluster.enabled"
#     value = "true"
#   }

#   set {
#     name  = "metrics.enabled"
#     value = "true"
#   }

#   set {
#     name  = "service.annotations.prometheus\\.io/port"
#     value = "9127"
#     type  = "string"
#   }
# }
