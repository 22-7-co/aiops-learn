import datetime
import os
from flask import Flask, request, jsonify
import numpy as np
import pandas as pd
import joblib
import requests

app = Flask(__name__)

# 加载模型和标准化器
model = joblib.load("time_qps_auto_scaling_model.pkl")
scaler = joblib.load("time_qps_auto_scaling_scaler.pkl")


# 定义函数从 Prometheus 获取 QPS
# TODO: 验证结果
def get_qps_from_prometheus():
    host = os.getenv(
        "PROMETHEUS_HOST",
        "prometheus-kube-prometheus-prometheus.prometheus.svc.cluster.local:9090",
    )
    url = f"http://{host}/api/v1/query"
    app.logger.info(url)
    # 训练数据是 10 分钟的 QPS，这里也取 10 分钟的 QPS
    query = 'rate(nginx_ingress_controller_nginx_process_requests_total{service="ingress-nginx-controller-metrics"}[10m])'
    response = requests.get(url, params={"query": query})
    results = response.json()
    app.logger.info(results)
    # 数据结构为 {data: {result: [{value: [timestamp, value]}]}}
    qps = float(results["data"]["result"][0]["value"][1])
    return qps


# 定义预测接口
@app.route("/predict", methods=["GET"])
def predict():
    try:
        qps = get_qps_from_prometheus()
        current_time = datetime.datetime.now().strftime("%H:%M:%S")
        # 时间处理
        minutes = (
            pd.to_datetime(current_time, format="%H:%M:%S").hour * 60
            + pd.to_datetime(current_time, format="%H:%M:%S").minute
        )
        sin_time = np.sin(2 * np.pi * minutes / 1440)
        cos_time = np.cos(2 * np.pi * minutes / 1440)

        # 特征向量
        data = {"QPS": [qps], "sin_time": [sin_time], "cos_time": [cos_time]}

        df = pd.DataFrame(data)
        features_scaled = scaler.transform(df)

        # 预测
        prediction = model.predict(features_scaled)
        # 为了避免实例数过大，这里限制最大实例数为 20
        if int(prediction[0]) > 20:
            return jsonify({"instances": 20})
        return jsonify({"instances": int(prediction[0])})
    except Exception as e:
        return jsonify({"error": str(e)})


# 运行服务
if __name__ == "__main__":
    app.run(debug=True, host="0.0.0.0", port=8080)
