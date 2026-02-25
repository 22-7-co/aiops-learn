#!/bin/bash

sudo apt-get update
sudo apt-get install ca-certificates curl gnupg -y
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg --yes
echo \
  "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo chmod a+r /etc/apt/keyrings/docker.gpg
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin golang-go jq -y

# clone kong
# git clone https://github.com/Kong/docker-kong.git
cd docker-compose
KONG_DATABASE=postgres sudo docker compose --profile database up -d

# 配置 AI 插件
# curl -X POST http://localhost:8001/services \
#   --data "name=ai-proxy" \
#   --data "url=http://localhost:32000"

# 创建 Router
# curl -X POST http://localhost:8001/services/ai-proxy/routes \
#   --data "name=chat" \
#   --data "paths[]=~/chat$"