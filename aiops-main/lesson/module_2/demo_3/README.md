# Install

https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli


# 腾讯云

## 获取密钥

进入右上角“访问管理”，点击左侧菜单“用户”，创建一个新用户，访问方式选择“编程访问”，创建成功后，会得到一个 SecretId 和 SecretKey。

## 配置密钥

```bash
$ export TENCENTCLOUD_SECRET_ID="secretid"
$ export TENCENTCLOUD_SECRET_KEY="secretkey"
$ export TENCENTCLOUD_REGION="ap-hongkong"
```

## 运行

1. 初始化

```bash
$ terraform init
```

2. 预览

```bash
$ terraform plan
```

3. 执行

```bash
$ terraform apply
```
