import torch
from transformers import AutoTokenizer, AutoModelForCausalLM

# Hugging Face 上的模型 ID，首次运行会自动下载到本地缓存（~/.cache/huggingface）
model_id = "Qwen/Qwen3-0.6B"

# 选择推理设备：优先 NVIDIA GPU (cuda)，其次 Apple 芯片 GPU (mps)，否则用 CPU
device = torch.device(
    "cuda" if torch.cuda.is_available()
    else "mps" if torch.backends.mps.is_available()
    else "cpu"
)
print(f"Using device: {device}")

# 分词器：负责文本 <-> token id 的相互转换，并提供对话模板
tokenizer = AutoTokenizer.from_pretrained(model_id)
# 因果语言模型（自回归，逐个预测下一个 token），加载后移动到目标设备
model = AutoModelForCausalLM.from_pretrained(model_id).to(device)

print("Model and tokenizer loaded successfully.")

# 对话历史：system 设定助手行为，user 是用户的提问
messages = [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "你好，请介绍自己"},
]

# 按模型自带的对话模板把 messages 拼成一段 prompt，并直接分词成张量
inputs = tokenizer.apply_chat_template(
    messages,
    add_generation_prompt=True,  # 末尾追加 "<|im_start|>assistant\n"，提示模型开始作答
    return_tensors="pt",         # 返回 PyTorch 张量
    return_dict=True,            # 返回 {"input_ids", "attention_mask"} 字典，便于 **inputs 解包
    enable_thinking=False,       # Qwen3 专有参数：关闭 <think> 思考过程，直接输出回答
).to(device)                     # 输入张量要和模型在同一设备上

# 打印拼接后的完整 prompt，便于观察对话模板的格式
print(f"Prompt: {tokenizer.decode(inputs['input_ids'][0])}")


# 生成回复：
#   max_new_tokens 最多新生成的 token 数
#   do_sample=True 启用随机采样（否则为贪心解码）
#   temperature    温度，越低输出越确定，越高越发散
#   top_p          核采样，只在累计概率前 top_p 的候选 token 中采样
# （0.7 / 0.8 是 Qwen3 非思考模式的官方推荐值）
generated = model.generate(**inputs, max_new_tokens=500, do_sample=True, temperature=0.7, top_p=0.8)


print(f"Original Generated response: {tokenizer.decode(generated[0], skip_special_tokens=True)}")

# generate 的输出 = prompt token + 新生成的 token，这里截掉 prompt 部分只保留回复
new_tokens = generated[0][inputs["input_ids"].shape[1]:]
# 解码为文本，skip_special_tokens 去掉 <|im_end|> 等特殊标记
print(f"Generated response: {tokenizer.decode(new_tokens, skip_special_tokens=True)}")
