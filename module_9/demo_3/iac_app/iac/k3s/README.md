# 快速开始

## 创建实验环境

1. 安装 [Terraform](https://learn.hashicorp.com/tutorials/terraform/install-cli)

1. 登录腾讯云，在“访问管理”模块中获取 `secret_id` 和 `secret_key`

1. 设置环境变量

```bash
export TF_VAR_secret_id=
export TF_VAR_secret_key=
```

1. 初始化 Terraform

```bash
terraform init
```
1. 查看执行计划

```bash
terraform plan
```

1. 执行

```bash
terraform apply -auto-approve
```

## 销毁实验环境

```bash
# 删除 k3s 状态，否则无法销毁
terraform state rm 'module.k3s'
terraform destroy -auto-approve
```