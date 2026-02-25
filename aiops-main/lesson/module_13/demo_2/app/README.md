# 初始化 golang-prometheus

```go mod init```

# Docker 镜像
```
docker.io/lyzhang1999/module13-go-prometheus-app
```

# Get Start

```
go mod tidy && go mod download
```

# 压力测试

```
hey -z 5m -q 5 -m GET -H "Accept: text/html" http://127.0.0.1:8080
```