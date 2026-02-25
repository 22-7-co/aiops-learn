# 使用方法

1. 设置环境变量

```bash
$ export TF_VAR_secret_id=
$ export TF_VAR_secret_key=
```

2. 初始化并应用（dev 环境）

```bash
cd dev
$ terraform init
$ terraform apply
```


## 访问 Argo CD
1. 设置 KUBECONFIG 环境变量
    
```bash
$ export KUBECONFIG="$(pwd)/config.yaml"
```

1. 获取密码

```bash
$ kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

1. 端口转发

```bash
$ kubectl port-forward svc/argocd-server -n argocd 8080:80
```
1. 访问 Dashboard

打开浏览器访问 http://127.0.0.1:8080，用户名 admin/密码为上面获取的密码。


## 销毁

```bash
# 先删除 k3s state 和 argocd state，否则会出错
$ terraform state rm 'helm_release.argo_cd'
$ terraform state rm 'module.k3s'

# 再执行删除
$ terraform destroy -auto-approve
```