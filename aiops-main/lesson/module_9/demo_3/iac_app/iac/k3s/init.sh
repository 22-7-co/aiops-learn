setup_cli() {
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
}

setup_helm_repo() {
    helm repo add grafana https://grafana.github.io/helm-charts
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
}

setup_kube_prometheus_grafana_loki() {
    helm upgrade -i loki -n monitoring --create-namespace grafana/loki-stack -f /tmp/values.yaml
    helm upgrade -i kube-prometheus-stack -n monitoring --create-namespace prometheus-community/kube-prometheus-stack --version "54.0.1" --set grafana.adminPassword=loki123
}

set_loki_nodeport() {
    export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
    kubectl patch service loki -n monitoring -p '{"spec":{"type":"NodePort","ports":[{"port":3100,"nodePort":31000}]}}'
    kubectl patch service kube-prometheus-stack-grafana -n monitoring -p '{"spec":{"type":"NodePort","ports":[{"port":80,"nodePort":31001}]}}'
}

main() {
    setup_cli
    setup_helm_repo
    setup_kube_prometheus_grafana_loki
    set_loki_nodeport
}

main