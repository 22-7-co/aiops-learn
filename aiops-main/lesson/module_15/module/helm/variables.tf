variable "filename" {
  default = ""
}

variable "kubernetes_ready" {
  default = ""
}

variable "helm_charts" {
  description = "An array of helm charts to deploy"
  type = list(object({
    name             = string
    repository       = string
    chart            = string
    namespace        = string
    create_namespace = bool
    values_file      = string
    version          = string
    set = list(object({
      name  = string
      value = string
    }))
  }))
  default = [
    {
      name             = "argocd"
      repository       = "https://argoproj.github.io/argo-helm"
      chart            = "argo-cd"
      namespace        = "argocd"
      create_namespace = true
      values_file      = ""
      version          = ""
      set = [
        {
          "name" : "",
          "value" : ""
        }
      ]
    },
  ]
}

variable "kubeconfig" {
  default = ""
}
