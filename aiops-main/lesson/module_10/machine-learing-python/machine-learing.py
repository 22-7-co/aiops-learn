import pandas as pd
import numpy as np
from sklearn.model_selection import train_test_split
from sklearn.linear_model import LinearRegression
from sklearn.metrics import mean_squared_error
from sklearn.preprocessing import StandardScaler
import joblib

# 加载数据
df = pd.read_csv("data.csv")

# 转换为从午夜开始的分钟数
df["minutes"] = (
    pd.to_datetime(df["timestamp"], format="%H:%M:%S").dt.hour * 60
    + pd.to_datetime(df["timestamp"], format="%H:%M:%S").dt.minute
)

# 将时间转换为正弦和余弦，以捕捉一天中时间的周期性
# 这里只考虑一天的训练数据，实际上你还可以考虑更多的时间特征，例如星期几（周末流量可能更大）、节假日、月份等因素
df["sin_time"] = np.sin(2 * np.pi * df["minutes"] / 1440)
df["cos_time"] = np.cos(2 * np.pi * df["minutes"] / 1440)

# 特征和目标变量
X = df[["QPS", "sin_time", "cos_time"]]
y = df["instances"]

# 分割数据为训练集和测试集
X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=0)

# 标准化特征
scaler = StandardScaler()
X_train_scaled = scaler.fit_transform(X_train)
X_test_scaled = scaler.transform(X_test)

# 创建并训练模型
model = LinearRegression()
model.fit(X_train_scaled, y_train)

# 模型评估
y_pred = model.coef_ * X_test_scaled + model.intercept_
mse = mean_squared_error(y_test, y_pred.sum(axis=1))

print("Model coefficients:", model.coef_)
print("Model intercept:", model.intercept_)
print("Mean Squared Error:", mse)

# 保存模型
joblib.dump(model, "time_qps_auto_scaling_model.pkl")

# 保存标准化器
joblib.dump(scaler, "time_qps_auto_scaling_scaler.pkl")
