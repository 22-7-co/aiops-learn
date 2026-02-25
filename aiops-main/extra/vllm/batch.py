from vllm import LLM, SamplingParams

prompts = [
    "你好，你是谁？",
    "你在干什么？",
    "你有智商吗？",
    "你 IQ 多少？",
]

sampling_params = SamplingParams(temperature=0.8, top_p=0.95, max_tokens=1024)

model_path = "/home/ubuntu/Qwen2.5-0.5B-Instruct"

llm = LLM(model=model_path, dtype="half")

outputs = llm.generate(prompts, sampling_params)

for output in outputs:
    prompt = output.prompt
    generated_text = output.outputs[0].text
    print(f"Prompt: {prompt!r}, Generated text: {generated_text!r}")
