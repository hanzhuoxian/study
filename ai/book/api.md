+# OpenAI Responses API 学习笔记

> 目标：通过可运行的 HTTP 和 Python 示例，理解 OpenAI Responses API 的请求结构、响应解析、多轮对话、流式输出和工具调用。
>
> 更新日期：2026-10-26。模型名称、参数支持和 SDK 方法会随版本变化；调用前请对照官方文档。下文用 `gpt-5.5` 作为示例模型名，请替换为你的项目和账号当前可用、并支持所需能力的模型。

## 目录

1. [准备工作](#1-准备工作)
2. [Responses API 概览](#2-responses-api-概览)
3. [最小请求：生成文本](#最小请求生成文本)
4. [输入与响应结构](#输入可以是什么)
5. [字段参考](#5-字段参考)
6. [多轮对话](#6-多轮对话)
7. [流式输出](#7-流式输出)
8. [结构化 JSON 输出](#8-结构化-json-输出)
9. [工具调用](#9-工具调用)
10. [常见问题与排错](#10-常见问题与排错)
11. [官方资料](#11-官方资料)

---

## 1. 准备工作

### 1.1 API Key

在终端设置密钥。不要把真实密钥写入代码、Git 仓库、前端网页或截图。

```bash
export OPENAI_API_KEY="你的_API_Key"
```

macOS/Linux 当前终端会话内有效。长期使用可配置到本机 shell 环境；生产服务则使用密钥管理系统或部署平台的 secret。不要把 API Key 放到浏览器端，因为用户可以查看并盗用它。

### 1.2 直接使用 HTTP

所有示例均使用 HTTPS 和 Bearer Token：

```http
Authorization: Bearer $OPENAI_API_KEY
Content-Type: application/json
```

请求发送到 `https://api.openai.com/v1` 下的接口。下面 curl 的环境变量展开由 shell 完成。

### 1.3 使用官方 SDK（Python）

```bash
python -m pip install --upgrade openai
```

```python
from openai import OpenAI

client = OpenAI()  # 默认读取 OPENAI_API_KEY
```

---

## 2. Responses API 概览

Responses API 使用 `/v1/responses` 端点，让请求输入和模型输出能够包含多个类型的内容与工具项。它支持普通文本、多模态输入、内置工具和自定义函数调用等能力（具体取决于模型）。

### 最小请求：生成文本

#### HTTP / curl

```bash
curl https://api.openai.com/v1/responses \\
  -H "Authorization: Bearer $OPENAI_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-5.5",
    "instructions": "你是一位简洁的中文编程老师。",
    "input": "用一句话解释什么是 HTTP API。"
  }'
```

#### Python SDK

```python
from openai import OpenAI

client = OpenAI()
response = client.responses.create(
    model="gpt-5.5",
    instructions="你是一位简洁的中文编程老师。",
    input="用一句话解释什么是 HTTP API。",
)
print(response.output_text)
```

Responses 的 HTTP 原始 JSON 中，模型生成内容位于 `output` 数组；Python SDK 另外提供 `response.output_text` 便于提取纯文本。详见字段参考。

### 输入可以是什么？

最简单时，`input` 是一个字符串，例如 `"把这句话翻译成英文：今天天气不错。"`。

需要角色、多个消息或多模态内容时，可以提供数组。以下是对应的 HTTP JSON 请求体片段：

```json
{
  "model": "gpt-5.5",
  "instructions": "请用中文回答。",
  "input": [
    {
      "role": "user",
      "content": [
        {"type": "input_text", "text": "请描述这张图片。"},
        {"type": "input_image", "image_url": "https://example.com/cat.jpg"}
      ]
    }
  ]
}
```

以上是图片 URL 的请求形状示例，实际使用时必须换成你有权访问的图片地址或合规的数据 URL，并确认所选模型支持图片输入。文件、音频等输入也有各自类型，不要把它们都当普通文本字符串处理。

#### 响应对象和 output items

Responses 原始响应的关键字段大致如下：

```json
{
  "id": "resp_example",
  "object": "response",
  "status": "completed",
  "model": "gpt-5.5",
  "output": [
    {
      "id": "msg_example",
      "type": "message",
      "role": "assistant",
      "status": "completed",
      "content": [
        {
          "type": "output_text",
          "text": "HTTP API 是程序通过 HTTP 请求与服务交换数据的一种接口。",
          "annotations": []
        }
      ]
    }
  ],
  "usage": {
    "input_tokens": 30,
    "output_tokens": 18,
    "total_tokens": 48
  }
}
```

关键点：

- 顶层 `output` 是数组，并非保证只有一条文本消息。
- 常见文本项结构是 `output[] -> type: "message" -> content[] -> type: "output_text"`。
- 也可能出现 `function_call`、内置工具调用、拒答标记、图片/音频生成等类型的 item。
- SDK 的 `response.output_text` 可以方便地收集文本；若需判断工具调用、状态或其他内容，请遍历 `response.output`。
- `status` 可能是 completed、incomplete、failed 等；不要只看有没有 response ID。
- `usage` 用 `input_tokens` / `output_tokens` 等字段统计，可通过这些字段查看本次请求的 token 用量。

## 5. 字段参考

本节集中解释 `POST /v1/responses` 的常用请求字段、输入 item、响应字段以及工具/结构化输出中的字段。API 的参数会随模型和版本变化；“可选”表示请求可以省略，并不表示每个模型都支持。应以 [Create a model response API Reference](https://developers.openai.com/api/reference/resources/responses/methods/create) 的当前定义为准。

### 5.1 顶层请求字段

| 字段 | 类型 / 是否必填 | 定义与用法 |
|---|---|---|
| `model` | string；必填 | 要调用的模型 ID。决定可用能力、上下文长度及支持参数。示例中的模型名只是占位，需替换为账号可用模型。 |
| `input` | string 或 input item 数组；通常必填 | 本次要模型处理的内容。可用简单字符串，也可用带 role、content 的消息及图片/文件/音频等内容项。 |
| `instructions` | string；可选 | 作为 system/developer 层级指令加入模型上下文，适合放任务规则和回答风格。使用 `previous_response_id` 时，前一轮的顶层 instructions 不会自动带到下一轮。 |
| `previous_response_id` | string；可选 | 指定前一 Response 的 ID，以继续其上下文。不能与 `conversation` 同时使用。 |
| `conversation` | string 或对象；可选 | 将本次交互关联到持久 Conversation；其 input/output items 会加入该 Conversation。与 `previous_response_id` 是不同的状态管理方式，不能同时传入。 |
| `max_output_tokens` | integer；可选 | 生成 token 上限，包含可见输出和推理 token。若达到限制，response 可能以 incomplete 状态结束。 |
| `temperature` | number；可选 | 控制采样随机性，范围及是否支持由模型决定。较高值更随机。通常不要同时调整它和 `top_p`。 |
| `top_p` | number；可选 | nucleus sampling 参数，限制模型从累计概率范围内选择 token。通常不要同时调整它和 `temperature`。 |
| `text` | object；可选 | 文本输出配置，包括 `format`（纯文本、JSON Schema 等）和 `verbosity`（支持的模型可选低、中、高详细度）。 |
| `tools` | array；可选 | 本次模型可以使用的工具定义，包括内置工具或应用自定义 function。声明工具不等于应用已执行自定义函数。 |
| `tool_choice` | string 或对象；可选 | 控制工具选择。常用 `auto`（模型决定）、`none`（不调用）、`required`（必须调用）；也可指定某个函数或工具，具体结构视工具类型而定。 |
| `parallel_tool_calls` | boolean；可选 | 是否允许模型并行发起多个工具调用。应用仍需逐个处理并按 call ID 回传结果。 |
| `max_tool_calls` | integer；可选 | 对单个 Response 中平台内置工具调用总数设上限；并非所有自定义函数流程都由平台执行。 |
| `stream` | boolean；可选，默认 false | 为 true 时通过 SSE 流式返回语义事件，而不是只返回最终 JSON。 |
| `store` | boolean；可选，默认 true（以当前文档为准） | 是否存储生成的 Response 以便后续 API 检索。按数据策略选择；设置 false 不会自动提供可用的持久 response 链。 |
| `metadata` | object；可选 | 附加到 Response 的键值元数据，适合内部标记和检索。文档限制最多 16 个键，键和值长度也有限。不要放秘密或敏感信息。 |
| `reasoning` | object；可选 | 推理模型的配置对象，常见子字段是 `effort`，可控制推理投入等级；支持值依模型变化。不要将其理解为请求模型展示私有推理过程。 |
| `service_tier` | string；可选 | 指定服务处理层级，如 auto/default/flex/priority 等；可用值与账户权限可能变化。 |
| `safety_identifier` | string；可选 | 稳定标识终端用户以支持滥用监测。官方建议使用哈希后的标识，避免发送邮箱等直接个人信息。 |
| `prompt_cache_key` | string；可选 | 为相似请求的提示缓存分桶提供稳定键；不要填入个人隐私。 |
| `background` | boolean；可选 | 请求以后台模式运行时使用；适合耗时任务，需配合查询状态等机制。不是普通同步请求必填字段。 |
| `include` | string 数组；可选 | 请求响应额外包含某些可选数据，例如引用或特定工具信息；只能使用参考文档列出的值。 |

### 5.2 input 消息与内容字段

| 字段 | 类型 / 是否必填 | 定义与用法 |
|---|---|---|
| `role` | string；消息必填 | 消息说话方，常见 `user`、`assistant`、`system`、`developer`。角色影响指令层级；普通终端输入用 `user`。 |
| `content` | string 或内容数组；消息必填 | 消息正文。纯文本可直接用字符串；混合文本、图片等时用内容对象数组。 |
| `type` | string；内容项必填 | 指示内容种类，如 `input_text`、`input_image`、`input_file`。类型和值结构必须匹配。 |
| `text` | string；文本内容必填 | `input_text` 项中的文本。 |
| `image_url` | string；图片输入可选其一 | 图片 URL 或 data URL；也可使用 `file_id` 引用已上传文件。图片细节级别可由该内容项的 `detail` 控制（如果模型支持）。 |
| `file_id` | string；文件/图片可选其一 | 已上传文件的 ID，用于引用平台文件。需保证文件权限和模型支持。 |
| `file_url` / `file_data` | string；按文件输入类型选用 | 通过 URL 或编码数据传文件。通常还需 `filename` 等字段，遵守具体输入类型 schema。不要把大型文件内容随意塞进 JSON。 |
| `detail` | string；可选 | 对支持的图片/文件视觉输入指定处理细节，如 auto/low/high；影响理解质量和输入成本，具体枚举按模型文档。 |
| `name` | string；可选 | 消息的可选名称标签，用于区分来源；不是权限或身份认证机制。 |

### 5.3 Response 响应字段

| 字段 | 类型 | 定义与用法 |
|---|---|---|
| `id` | string | 本次 Response 的唯一 ID，可用于续聊、检索或日志关联。 |
| `object` | string | 对象类型标记，Response 对象为 `response`。 |
| `created_at` | number | 创建时间的 Unix 时间戳（秒）。 |
| `status` | string | 生命周期状态，可能为 queued、in_progress、completed、incomplete、failed、cancelled 等。先看状态再判断内容是否完整。 |
| `error` | object 或 null | 请求处理失败时的错误信息；成功响应通常为 null。 |
| `incomplete_details` | object 或 null | Response 未完整完成时的原因，例如达到输出 token 上限。 |
| `model` | string | 实际用于生成本次响应的模型标识。 |
| `output` | array | 模型返回的有序 output items。可能有 message、function_call、reasoning 或其他类型；不要假定第一个 item 就是文本。 |
| `output_text` | SDK 便捷属性 | 官方 SDK 从输出消息中提取文本的便利属性；这是 SDK helper，不是 REST JSON 保证存在的顶层字段。 |
| `usage` | object 或 null | token 用量。常见包括 `input_tokens`、`output_tokens`、`total_tokens` 及细分项。 |
| `previous_response_id` | string 或 null | 本 Response 关联的前一 Response ID。 |
| `conversation` | object 或 null | 与本 Response 关联的 Conversation 信息。 |
| `metadata` | object | 创建时附加的应用元数据。 |

#### output message 的字段

| 字段 | 定义 |
|---|---|
| `type` | item 类型。普通助手消息为 `message`；工具请求可能是 `function_call`。 |
| `id` | output item 的唯一标识。 |
| `role` | 消息角色，通常为 `assistant`。 |
| `status` | 该 item 的状态，如 completed 或 in_progress。 |
| `content` | 消息内容数组，可包含文本或拒答等内容项。 |
| `content[].type` | 内容类型，常见 `output_text` 或 `refusal`。 |
| `content[].text` | 输出文本内容。 |
| `content[].annotations` | 与输出相关的引用或注释，例如 web/file citation；没有注释时通常为空数组。 |

#### usage 的字段

| 字段 | 定义 |
|---|---|
| `input_tokens` | 输入侧 token 总数。 |
| `output_tokens` | 输出侧 token 总数，可能包含推理 token。 |
| `total_tokens` | 输入与输出合计。 |
| `input_tokens_details.cached_tokens` | 输入中由缓存命中的 token 数（返回时可能为 0）。 |
| `output_tokens_details.reasoning_tokens` | 输出 token 中归类为推理用量的数量（可能为 0）。 |

### 5.4 Structured Outputs 字段

| 字段 | 定义 |
|---|---|
| `text.format` | 指定文本生成格式的配置对象。 |
| `text.format.type` | `text` 表示普通文本；`json_schema` 表示按给定 schema 生成结构化 JSON；`json_object` 是较旧的 JSON mode，支持时优先选 schema。 |
| `text.format.name` | schema 格式的名称，便于识别；使用合法短名称。 |
| `text.format.schema` | JSON Schema 对象，定义 JSON 的字段、类型与约束。 |
| `text.format.strict` | 是否严格遵循 schema。设为 true 时应使用 API 支持的 schema 子集。 |
| `text.verbosity` | 可选的文字详细程度设置；low/medium/high 的支持依模型而定。 |

### 5.5 自定义 function 工具字段

| 字段 | 定义 |
|---|---|
| `tools[].type` | 函数工具使用 `function`；内置工具用各自类型，如 web search/file search。 |
| `tools[].name` | 函数名，模型返回调用时应用据此路由到正确函数。 |
| `tools[].description` | 告诉模型何时以及如何使用函数；描述应明确、避免模糊。 |
| `tools[].parameters` | JSON Schema 对象，定义函数参数。 |
| `tools[].strict` | 启用严格参数 schema 遵循。严格模式下对象需 `additionalProperties: false`，字段通常都列入 required；可用 nullable 类型表达可选值。 |
| `output[].call_id` | 函数调用的关联 ID。应用提交结果时必须原样放进 `function_call_output.call_id`。 |
| `output[].arguments` | 模型生成的 JSON 编码参数字符串；应用应 JSON 解析并做业务校验。 |
| `function_call_output.output` | 应用执行函数后返回给模型的结果，通常是字符串；可以是 JSON 字符串。 |

工具 schema 字段完整定义与内置工具字段不同；以上是自定义 function 入门所需字段。每种内置工具都有自己的参数表。

## 6. 多轮对话

### 6.1 多轮对话方式一：previous_response_id

如果希望 API 关联前一次响应，可将第一轮的 ID 放在第二轮的 `previous_response_id` 中：

```python
from openai import OpenAI

client = OpenAI()

first = client.responses.create(
    model="gpt-5.5",
    instructions="你是耐心的 Python 老师。",
    input="什么是列表？",
)
print("第一轮：", first.output_text)

second = client.responses.create(
    model="gpt-5.5",
    previous_response_id=first.id,
    input="它和元组有什么区别？",
)
print("第二轮：", second.output_text)
```

要点：

1. 把 `first.id` 当成 opaque ID 使用，不要自行拼接或推导。
2. 后续请求仍要带 `model`。
3. **不要假设前一请求的 `instructions` 会自动沿用。** 如有需要，每一轮显式发送本轮的 instructions。
4. 关联响应会把之前的上下文纳入续聊，但这不代表输入上下文没有 token 用量；仍需考虑上下文长度和成本。
5. 如果不想依赖响应链，也可以在自己的应用中保存完整 item 历史，并在后续 HTTP/Python 请求中重新传入。

### 6.2 多轮对话方式二：自己保存 input / output items

需要显式控制上下文时，可以把上一轮输入和输出 item 接到下一轮 input：

HTTP 请求体中的 `input` 是数组；把前一次响应的 `output` 数组原样放入下一次请求，再追加新问题。保留模型返回的完整相关 items，对涉及工具或推理模型的流程尤其重要，不要只复制可见文本而丢掉续接所需 items。

## 7. 流式输出

Responses 流式接口同样使用 `stream: true`，但事件带有明确的 `type`。常见文字事件包括 `response.output_text.delta` 和结束事件 `response.completed`。

```bash
curl -N https://api.openai.com/v1/responses \\
  -H "Authorization: Bearer $OPENAI_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-5.5","input":"用三句话解释数据库索引。","stream":true}'
```

Python SDK：

```python
from openai import OpenAI

client = OpenAI()
stream = client.responses.create(
    model="gpt-5.5",
    input="用三句话解释数据库索引。",
    stream=True,
)
for event in stream:
    if event.type == "response.output_text.delta":
        print(event.delta, end="", flush=True)
    elif event.type == "response.completed":
        print()
```

Responses 的语义事件可以让应用区分 response 创建、文本增量、item 完成、工具调用以及最终完成/失败。只需显示文本时筛选文本 delta；需要工具流程时还要监听相应的 item 和函数调用事件。

## 8. 结构化 JSON 输出

如果程序要稳定消费 JSON，优先要求符合 JSON Schema 的 Structured Outputs，而不是只在提示词里说“请输出 JSON”。

```python
from openai import OpenAI

client = OpenAI()
response = client.responses.create(
    model="gpt-5.5",
    input="给一本书生成标题、作者和 1 到 3 个标签。书名：Python 入门",
    text={
        "format": {
            "type": "json_schema",
            "name": "book_metadata",
            "strict": True,
            "schema": {
                "type": "object",
                "properties": {
                    "title": {"type": "string"},
                    "author": {"type": "string"},
                    "tags": {"type": "array", "items": {"type": "string"}}
                },
                "required": ["title", "author", "tags"],
                "additionalProperties": False
            }
        }
    }
)

print(response.output_text)  # 输出为符合 schema 的 JSON 文本
```

严格 schema 通常要求列出所有 required 字段，并为对象设置 `additionalProperties: false`。仍应处理 refusal、输出不完整和 API 错误等情况。各模型支持的 schema 子集可能有限。

---

## 9. 工具调用

工具调用是学习 API 与应用程序协作的关键：

1. 应用向模型发送工具的名称、描述和参数 schema。
2. 模型在需要时返回工具调用请求和参数。
3. 应用检查并执行函数。
4. 应用把工具结果关联到相应调用，再发给模型。
5. 模型根据结果组织最终答复。

模型并没有直接执行本地 Python 函数；第 3 步由你的程序负责。生产应用要校验参数、控制权限，不能把模型输出当可信命令。

### 9.1 完整的 Python 天气函数调用

这是一个演示“城市天气查询”的假函数。实际项目应把 `get_weather` 换成真正的、由服务端控制的天气 API 调用。

```python
import json
from openai import OpenAI

client = OpenAI()

tools = [{
    "type": "function",
    "name": "get_weather",
    "description": "查询指定城市的天气。",
    "parameters": {
        "type": "object",
        "properties": {"city": {"type": "string", "description": "城市名"}},
        "required": ["city"],
        "additionalProperties": False,
    },
    "strict": True,
}]

def get_weather(city: str) -> dict:
    # 教学用模拟结果；实际程序在这里调用你信任的天气服务。
    return {"city": city, "temperature_c": 18, "condition": "晴"}

response = client.responses.create(
    model="gpt-5.5",
    input="巴黎现在天气怎么样？",
    tools=tools,
)

# 有时模型会直接回答；有时会请求一个或多个函数。
while True:
    calls = [item for item in response.output if item.type == "function_call"]
    if not calls:
        print(response.output_text)
        break

    tool_outputs = []
    for call in calls:
        if call.name != "get_weather":
            raise ValueError(f"未知工具：{call.name}")
        args = json.loads(call.arguments)
        result = get_weather(args["city"])
        tool_outputs.append({
            "type": "function_call_output",
            "call_id": call.call_id,
            "output": json.dumps(result, ensure_ascii=False),
        })

    # 保留模型上轮的完整 output items，并补上每个调用的结果。
    response = client.responses.create(
        model="gpt-5.5",
        previous_response_id=response.id,
        input=tool_outputs,
        tools=tools,
    )
```

解释：

- `tools` 中声明函数 schema，模型会根据描述决定是否调用。
- `function_call.arguments` 是 JSON 编码字符串，需要解析并验证。
- `call_id` 用于把你返回的 `function_call_output` 对应到正确调用。
- 可能同时出现多个调用，因此代码按列表处理。
- 工具结果会成为模型后续输入，模型再生成自然语言答复。
- 循环要设置最大轮次、超时与错误处理，防止失控；此入门代码省略了生产级重试与日志。

## 10. 常见问题与排错

### 模型与能力选择

- **选择模型：** 先核实模型支持的输入模态、工具和请求参数。Responses API 支持的能力取决于所选模型。
- **要用特定模型/参数：** 先核实此模型是否支持该端点与功能；不要只看 endpoint 名称猜测。

### 常见错误

| 现象 | 排查方向 |
|---|---|
| 401 / 认证失败 | 检查环境变量是否设置、key 是否正确、shell 是否展开变量；不要打印密钥来排查 |
| 403 / 权限不足 | 检查项目、组织、模型访问权限及账户状态 |
| 404 / 找不到模型或路径 | 检查 endpoint 拼写、模型名称是否正确和可用 |
| 400 / 参数不支持 | 逐项检查目标模型支持的参数名称、类型和取值范围 |
| `output_text` 为空 | 检查 `response.output` 是否包含工具调用、拒答或非文本模态 item |
| 输出被截断 | 检查 finish reason / response status、token 上限和上下文长度 |
| 多轮不记得前文 | 确认 previous_response_id、conversation 或自行维护的 input/output items 使用正确 |
| 工具一直没执行 | 模型只是给出调用请求；检查应用是否解析调用、执行函数并回传结果 |
| SSE 解析失败 | 按完整事件边界读取；事件可能被网络分块，不能假定一次读到一行或一个完整 JSON |

API 出错时优先查看 HTTP 状态码、错误对象中的 message/param/type，以及请求 ID；记录日志时注意脱敏用户输入和密钥。

---

## 11. 官方资料

- [Responses API 概览](https://developers.openai.com/api/docs/guides/responses)
- [Responses API：Create a model response](https://developers.openai.com/api/reference/resources/responses/methods/create)
- [文本生成指南](https://developers.openai.com/api/docs/guides/text)
- [流式响应](https://developers.openai.com/api/docs/guides/streaming-responses)
- [Function calling 指南](https://developers.openai.com/api/docs/guides/function-calling)
- [Structured Outputs 指南](https://developers.openai.com/api/docs/guides/structured-outputs)

> 学习路径建议：先运行最小问答 → 查看并解析响应 → 实现多轮上下文 → 改为流式 → 最后增加结构化输出和工具调用。每一步都使用服务端密钥，并记录自己发送的输入字段与拿到的响应形状。

