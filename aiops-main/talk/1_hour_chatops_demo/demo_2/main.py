from pydantic import BaseModel
from openai import OpenAI

client = OpenAI()


class ServiceAction(BaseModel):
    service_name: str
    action: str


completion = client.beta.chat.completions.parse(
    model="gpt-4o-2024-08-06",
    messages=[
        {
            "role": "system",
            "content": "解析内容，并提取对象, action 可以是 get_log（获取日志）、restart（重启服务）、delete（删除工作负载）",
        },
        {
            "role": "user",
            "content": "帮我重启 payment 服务",
        },
    ],
    response_format=ServiceAction,
)


event = completion.choices[0].message.parsed
print(event.service_name)
print(event.action)
