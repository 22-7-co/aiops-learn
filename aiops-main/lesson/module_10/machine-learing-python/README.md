## 部署 kube-prometheus-stack
```bash
helm upgrade prometheus prometheus-community/kube-prometheus-stack \
--namespace prometheus  \
--set prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues=false \
--set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
--install --create-namespace
```

## 部署 ingress-nginx
```bash
helm upgrade ingress-nginx ingress-nginx/ingress-nginx \
--namespace ingress-nginx \
--set controller.metrics.enabled=true \
--set controller.metrics.serviceMonitor.enabled=true \
--set controller.metrics.serviceMonitor.additionalLabels.release="prometheus" \
--install --create-namespace
```

## 检查部署状态
```bash
kubectl port-forward svc/prometheus-kube-prometheus-prometheus -n prometheus 9090:9090
```
打开浏览器访问 http://localhost:9090


## 构建并运行镜像

```bash
docker build -t docker.io/library/machine-learning-python .
```

1. 将镜像载入 Kind 集群

```bash
kind load docker-image docker.io/library/machine-learning-python
```

2. 部署推理服务

```bash
kubectl apply -f python-deployment.yaml
```

3. 部署示例 nginx deployment
```bash
kubectl apply -f nginx-deployment.yaml
```
这会部署 nginx，同时启动一个容器不断请求 nginx-ingress 以模拟更高的 QPS。