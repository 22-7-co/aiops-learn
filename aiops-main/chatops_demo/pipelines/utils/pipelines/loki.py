import requests
import time
import os
import yaml

current_path = os.path.dirname(os.path.abspath(__file__))
config_file_path = os.path.join(
    current_path, "..", "..", "..", "resource", "config.yaml"
)

with open(config_file_path, "r") as file:
    config = yaml.safe_load(file)

server_url = config["clusters"][0]["cluster"]["server"]
server_url = server_url.replace("https", "http")
server_url_without_port = server_url.rsplit(":", 1)[0]

loki_url = f"{server_url_without_port}:31000/loki/api/v1/query_range"


def get_logs_from_loki(query, start_time, end_time):
    """从 Loki 获取日志"""
    # print(f"Start Time: {start_time}, End Time: {end_time}")

    # 请求参数
    params = {
        "query": query,
        "start": str(start_time),
        "end": str(end_time),
        "limit": 10,
    }

    # 发送 GET 请求
    response = requests.get(loki_url, params=params)

    # 处理响应
    if response.status_code == 200:
        data = response.json()
        logs = []
        for stream in data.get("data", {}).get("result", []):
            for value in stream.get("values", []):
                timestamp, log = value
                logs.append(f"{timestamp}: {log}")
        return logs
    else:
        print(f"Error: {response.status_code}, {response.text}")
        return []


def get_cluster_ip_address():
    current_path = os.path.dirname(os.path.abspath(__file__))
    config_file_path = os.path.join(
        current_path, "..", "..", "..", "resource", "config.yaml"
    )

    with open(config_file_path, "r") as file:
        config = yaml.safe_load(file)

    server_url = config["clusters"][0]["cluster"]["server"]
    server_url = server_url.replace("https", "http")
    server_url_without_port = server_url.rsplit(":", 1)[0]
    return server_url_without_port
