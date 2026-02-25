import openai

client = openai.OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="apikey",
)

completion = client.chat.completions.create(
    model="gpt-3.5-turbo",
    messages=[
        {
            "role": "system",
            "content": "你是一个 AI 助手",
        },
        {"role": "user", "content": "你是谁？"},
    ],
)

print(completion.choices[0].message)
