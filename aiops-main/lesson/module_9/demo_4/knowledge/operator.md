# 内部运维知识库和解决方案手册

1. Database connection failed：数据库连接失败，请检查数据库是否正常运行，数据库连接配置是否正确（账号密码）
2. Service 500 Error：下游服务 500 错误，请找对应的服务负责人：
    1. order-processing 服务：小王
    2. payment-processing 服务：小李
    3. user-processing 服务：小张
3. Memory OOM：内存溢出，请检查应用内存使用情况，是否存在内存泄漏，如有必要，请联系小王提高容器内存限制
4. Fraud detection failed Payment flagged as potentially fraudulent：欺诈检测失败，请联系小李检查欺诈检测服务是否正常
