from openai import OpenAI
import json
from loki import get_logs_from_loki
import time

client = OpenAI()


def analyze_loki_log(query_str, start_time, end_time):
    print("Get log from Loki: ", query_str, start_time, end_time)
    logs = get_logs_from_loki(query_str, start_time, end_time)
    return json.dumps(logs)


def get_start_end_time(code):
    local_vars = {}
    exec(code, {}, local_vars)
    start_time = local_vars.get("start_time")
    end_time = local_vars.get("end_time")
    # Fix: end_time should be current time
    end_time = str(time.time() * 1e9)
    start_time = "{:.0f}".format(float(start_time))
    end_time = "{:.0f}".format(float(end_time))
    print(start_time, end_time)
    return json.dumps({"start_time": start_time, "end_time": end_time})


def run_conversation():
    """分析最近 1 天 app=grafana 且关键字包含 Error 的日志"""

    # 步骤一：把所有预定义的 function 传给 chatgpt
    query = input("输入查询指令：")
    messages = [
        {
            "role": "system",
            "content": "你是一个 Loki 日志分析助手，你可以帮助用户分析 Loki 日志，你可以调用多个函数来帮助用户完成任务，并最终尝试分析错误产生的原因",
        },
        {
            "role": "user",
            "content": query,
        },
    ]
    tools = [
        {
            "type": "function",
            "function": {
                "name": "analyze_loki_log",
                "description": "从 Loki 获取日志，如果缺少时间戳，可以使用 get_start_end_time 工具获取时间戳",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "query_str": {
                            "type": "string",
                            "description": 'Loki 查询字符串，例如：{app="grafana"} |= "Error"',
                        },
                        "start_time": {
                            "type": "string",
                            "description": "查询开始时间，纳秒时间戳",
                        },
                        "end_time": {
                            "type": "string",
                            "description": "查询结束时间，纳秒时间戳",
                        },
                    },
                    "required": ["query_str", "start_time", "end_time"],
                },
            },
        },
        {
            "type": "function",
            "function": {
                "name": "get_start_end_time",
                "description": "获取开始和结束时间的时间戳字符串（纳秒格式）",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "code": {
                            "type": "string",
                            "description": "用来获取开始时间和结束时的 Python 代码，endtime 取当前时间的时间戳，将时间戳保存在 start_time 和 end_time 两个字符串变量，格式示例 1721115879758772992",
                        },
                    },
                    "required": ["code"],
                },
            },
        },
    ]
    i = 0
    print("开始思考链式方法调用......")
    while True:
        i += 1
        response = client.chat.completions.create(
            model="gpt-4o",
            messages=messages,
            tools=tools,
            tool_choice="auto",
        )
        response_message = response.choices[0].message
        tool_calls = response_message.tool_calls
        # 步骤二：检查 LLM 是否调用了 function
        if tool_calls is None:
            # 结束对 tools 的循环调用
            break
        if tool_calls:
            available_functions = {
                "analyze_loki_log": analyze_loki_log,
                "get_start_end_time": get_start_end_time,
            }
            messages.append(response_message)
            # 步骤三：把每次 function 调用和返回的信息传给 model
            for tool_call in tool_calls:
                function_name = tool_call.function.name
                print("第", i, "轮调用", function_name, "函数")
                function_to_call = available_functions[function_name]
                function_args = json.loads(tool_call.function.arguments)
                function_response = function_to_call(**function_args)
                print("第", i, "轮函数返回结果", function_response)
                print("进行下一轮思考......\n\n")
                messages.append(
                    {
                        "tool_call_id": tool_call.id,
                        "role": "tool",
                        "name": function_name,
                        "content": function_response,
                    }
                )
    print("链式函数调用结束，开始分析日志......")
    final_response = client.chat.completions.create(
        model="gpt-4o",
        messages=messages,
    )
    return final_response.choices[0].message.content


print(run_conversation())
