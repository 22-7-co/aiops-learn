from openai import OpenAI

client = OpenAI(
    api_key="sk-T6SlqfUnyFytejvA3c1584F87d6343878232185e26243b1d",
    base_url="https://api.apiyi.com/v1",
)

completion = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[
        {
            "role": "system",
            "content": "将以下英文翻译成中文",
        },
        {
            "role": "user",
            "content": "Thoughts on a Quiet Night\nThe moonlight shines before my bed, like frost upon the ground.\nI look up to see the moon, then lower my gaze, overwhelmed by homesickness.",
        },
    ],
)

print(completion.choices[0].message.content)
