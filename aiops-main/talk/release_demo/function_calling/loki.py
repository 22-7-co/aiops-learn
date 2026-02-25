import requests
import time

# Loki URL, CHANGE IT!
loki_url = "http://43.154.98.158:31000/loki/api/v1/query_range"


def get_logs_from_loki(query, start_time, end_time):
    """从 Loki 获取日志"""
    # print(f"Start Time: {start_time}, End Time: {end_time}")

    # 请求参数
    params = {
        "query": query,
        "start": str(start_time),
        "end": str(end_time),
        "limit": 3,
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
