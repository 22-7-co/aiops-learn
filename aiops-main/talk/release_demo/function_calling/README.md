# 快速开始

1. 设置 OPENAI API Key

    ```bash
    export OPENAI_API_KEY=
    ```

1. 安装依赖
    
    ```bash
    pip install requests openai
    ```

1. 修改 loki.py 第 5 行，设置为 K3s 部署的 Loki 的 URL
    
        ```python
        loki_url = "http://101.32.77.237:31000/loki/api/v1/query_range"
        ```
        
1. 尝试 Function Calling 调用
    
    ```bash
    python function_calling.py
    ```