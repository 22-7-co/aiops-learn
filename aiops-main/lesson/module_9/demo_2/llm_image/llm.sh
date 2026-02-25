#!/bin/bash
export OLLAMA_HOST=0.0.0.0:8080
export OLLAMA_MODELS=/usr/share/ollama/.ollama/models

if [ ! -f /usr/local/bin/ollama ]; then 
  curl -fsSL https://ollama.com/install.sh | sh
else
  echo "ollama 已安装"
fi

sudo mkdir -p /usr/share/ollama/.ollama/models
sudo useradd -r -s /bin/false -U -m -d /usr/share/ollama ollama
sudo usermod -a -G ollama $(whoami)
sudo chown -R ollama:ollama /usr/share/ollama
sudo chmod -R 755 /usr/share/ollama

service_content="[Unit]
Description=Ollama Service
After=network-online.target

[Service]
Environment=\"OLLAMA_HOST=0.0.0.0:8080\"
Environment=\"OLLAMA_MODELS=/usr/share/ollama/.ollama/models\"
ExecStart=/usr/local/bin/ollama serve
User=ollama
Group=ollama
Restart=always
RestartSec=3
Environment=\"PATH=\$PATH\"

[Install]
WantedBy=default.target
"

file_path="/etc/systemd/system/ollama.service"
echo "$service_content" | sudo tee "$file_path" > /dev/null

sudo systemctl daemon-reload
sudo systemctl enable ollama

echo "Ollama 服务就绪"

sleep 10

echo "Ollama 预加载模型"
OLLAMA_HOST=0.0.0.0:8080 ollama pull qwen2:0.5b
