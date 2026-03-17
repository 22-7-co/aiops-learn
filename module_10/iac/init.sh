setup_cli() {
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
}

setup_helm_repo() {
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
}

setup_kube_prometheus_grafana_loki() {
    helm upgrade -i kube-prometheus-stack -n monitoring --create-namespace prometheus-community/kube-prometheus-stack --version "54.0.1" --set grafana.adminPassword=loki123 --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false --set prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues=false
}

set_nodeport() {
    export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
    kubectl patch service kube-prometheus-stack-grafana -n monitoring -p '{"spec":{"type":"NodePort","ports":[{"port":80,"nodePort":31001}]}}'
}

install_nginx_ingress() {
    helm upgrade --install ingress-nginx ingress-nginx \
    --repo https://kubernetes.github.io/ingress-nginx \
    --namespace ingress-nginx \
    --create-namespace \
    --set controller.metrics.enabled=true \
    --set controller.metrics.serviceMonitor.enabled=true \
    --set controller.metrics.serviceMonitor.additionalLabels.release=kube-prometheus-stack
}

main() {
    setup_cli
    setup_helm_repo
    setup_kube_prometheus_grafana_loki
    set_nodeport
    install_nginx_ingress
}

main