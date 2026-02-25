from typing import List, Optional
from pydantic import BaseModel
from schemas import OpenAIChatMessage
import os
import requests
import json

from utils.pipelines.main import (
    get_last_user_message,
    add_or_update_system_message,
    get_tools_specs,
)


class Pipeline:
    class Valves(BaseModel):
        # List target pipeline ids (models) that this filter will be connected to.
        # If you want to connect this filter to all pipelines, you can set pipelines to ["*"]
        pipelines: List[str] = []

        # Assign a priority level to the filter pipeline.
        # The priority level determines the order in which the filter pipelines are executed.
        # The lower the number, the higher the priority.
        priority: int = 0

        # Valves for function calling
        OPENAI_API_BASE_URL: str
        OPENAI_API_KEY: str
        TASK_MODEL: str
        TEMPLATE: str

    def __init__(self):
        # Pipeline filters are only compatible with Open WebUI
        # You can think of filter pipeline as a middleware that can be used to edit the form data before it is sent to the OpenAI API.
        self.type = "filter"

        # Optionally, you can set the id and name of the pipeline.
        # Best practice is to not specify the id so that it can be automatically inferred from the filename, so that users can install multiple versions of the same pipeline.
        # The identifier must be unique across all pipelines.
        # The identifier must be an alphanumeric string that can include underscores or hyphens. It cannot contain spaces, special characters, slashes, or backslashes.
        # self.id = "function_calling_blueprint"
        self.name = "Function Calling Blueprint"

        # Initialize valves
        self.valves = self.Valves(
            **{
                "pipelines": ["*"],  # Connect to all pipelines
                "OPENAI_API_BASE_URL": os.getenv(
                    "OPENAI_API_BASE_URL", "https://api.openai.com/v1"
                ),
                "OPENAI_API_KEY": os.getenv("OPENAI_API_KEY", "YOUR_OPENAI_API_KEY"),
                "TASK_MODEL": os.getenv("TASK_MODEL", "gpt-4o"),
                "TEMPLATE": """Base on Function calling result inside <context></context> XML tags.
<context>
    {{CONTEXT}}
</context>

When answer to user:
- If you don't know, just say that you don't know.
- If you don't know when you are not sure, ask for clarification.
Avoid mentioning that you obtained the information from the context.
And answer according to the language of the user's question.""",
            }
        )

    async def on_startup(self):
        # This function is called when the server is started.
        print(f"on_startup:{__name__}")
        pass

    async def on_shutdown(self):
        # This function is called when the server is stopped.
        print(f"on_shutdown:{__name__}")
        pass

    def convert_type(self, properties):
        for key, value in properties.items():
            if value.get("type") == "str":
                value["type"] = "string"
            if value.get("type") == "literal":
                value["type"] = "string"
            if value.get("type") == "int":
                value["type"] = "number"
        return properties

    async def inlet(self, body: dict, user: Optional[dict] = None) -> dict:
        # If title generation is requested, skip the function calling filter
        if body.get("title", False):
            return body

        print(f"pipe:{__name__}")
        print(user)

        # Get the last user message
        user_message = get_last_user_message(body["messages"])

        # Get the tools specs
        tools_specs = get_tools_specs(self.tools)

        # System prompt for function calling
        fc_system_prompt = (
            f"Tools: {json.dumps(tools_specs, indent=2)}"
            + """
If a function tool doesn't match the query, return an empty string. Else, pick a function tool, fill in the parameters from the function tool's schema, and return it in the format { "name": \"functionName\", "parameters": { "key": "value" } }. Only pick a function if the user asks.  Only return the object. Do not return any other text."
"""
        )

        # rebuild function calling list
        restructured_tools = [
            {
                "type": "function",
                "function": {
                    "name": tool["name"],
                    "description": tool["description"],
                    "parameters": {
                        "type": tool["parameters"]["type"],
                        "properties": self.convert_type(
                            tool["parameters"]["properties"]
                        ),
                        "required": tool["parameters"]["required"],
                        "additionalProperties": False,
                    },
                },
            }
            for tool in tools_specs
        ]

        print("Tools list: ", json.dumps(restructured_tools, indent=2))

        new_messages = [
            {
                "role": "system",
                "content": "You are a helpful customer support assistant. Use the supplied tools to assist the user.",
            }
        ]

        # 用新的系统提示语和用户消息组装新的消息列表
        new_messages = new_messages + body["messages"]
        # print(new_messages)

        r = None
        try:
            # Call the OpenAI API to get the function response
            r = requests.post(
                url=f"{self.valves.OPENAI_API_BASE_URL}/chat/completions",
                json={
                    "model": self.valves.TASK_MODEL,
                    "tools": restructured_tools,
                    "messages": new_messages,
                    # TODO: dynamically add response_format?
                    # "response_format": {"type": "json_object"},
                },
                headers={
                    "Authorization": f"Bearer {self.valves.OPENAI_API_KEY}",
                    "Content-Type": "application/json",
                },
                stream=False,
            )
            r.raise_for_status()

            response = r.json()
            print("\nLLM Response: ", response, "\n")
            response_message = response["choices"][0]["message"]
            tool_calls = response_message["tool_calls"]
            if tool_calls is None:
                return
            # 可能返回多个 tools 链式调用，所以要遍历
            for tool_call in tool_calls:
                function_name = tool_call["function"]["name"]
                function = getattr(self.tools, function_name)
                try:
                    function_args = json.loads(tool_call["function"]["arguments"])
                except Exception as e:
                    print(e)
                    function_args = {}
                function_response = function(**function_args)
                print("\nFunction 调用结果: ", function_response, "\n")
                # 按照 openai 的文档，必须要把 function 调用结果添加到 new_messages 中，否则会导致后续的消息处理出错
                new_messages.append(
                    {
                        "tool_call_id": tool_call["id"],
                        "role": "tool",
                        "name": function_name,
                        "content": function_response,
                    }
                )
                # 获得函数调用之后，需要再进行一次独立的基于函数调用结果的问答推理，这里把函数调用结果放到 context 中，然后返回
                system_prompt = self.valves.TEMPLATE.replace(
                    "{{CONTEXT}}", function_response
                )
                messages = add_or_update_system_message(system_prompt, body["messages"])
                # print("messages: ", messages)
                # Return the updated messages
            return {**body, "messages": messages}

            # NNNNNNNNNot loger needed
            # Parse the function response
            content = response["choices"][0]["message"]["content"]
            if content != "":
                result = json.loads(content)
                print(result)

                # Call the function
                if "name" in result:
                    function = getattr(self.tools, result["name"])
                    function_result = None
                    try:
                        function_result = function(**result["parameters"])
                    except Exception as e:
                        print(e)

                    # Add the function result to the system prompt
                    if function_result:
                        system_prompt = self.valves.TEMPLATE.replace(
                            "{{CONTEXT}}", function_result
                        )

                        print(system_prompt)
                        messages = add_or_update_system_message(
                            system_prompt, body["messages"]
                        )

                        # Return the updated messages
                        return {**body, "messages": messages}

        except Exception as e:
            print(f"Error: {e}")
            print(r.json())
            if r:
                try:
                    print(r.json())
                except:
                    pass

        return body
