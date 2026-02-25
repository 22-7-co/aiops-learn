---
name: k8s-deploy
description: 生成 K8s YAML 并部署资源
---

# 部署流程

1. 根据用户需求生成完整的 K8s YAML
2. **必须调用 DeployResource 工具**，传入生成的 YAML
3. 返回部署结果
