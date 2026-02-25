terraform {
  required_version = "> 0.13.0"
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "0.2.1"
    }
  }
}

provider "kind" {}

# Create a cluster
resource "kind_cluster" "default" {
  name = "test-cluster"
}

provider "helm" {
  kubernetes {
    config_path = "test-cluster-config"
  }
}

resource "helm_release" "crossplane" {
  depends_on = [kind_cluster.default]
  name       = "crossplane"

  repository       = "https://charts.crossplane.io/stable"
  chart            = "crossplane"
  namespace        = "crossplane-system"
  create_namespace = true
}
