import json
from collections.abc import Callable
from datetime import date

from llm.llm import LLM
from search.search import search


def get_current_date() -> str:
    """返回今天的日期，帮助模型判断什么是"最新"。"""
    return date.today().isoformat()


# 工具定义：用 JSON Schema 描述参数，模型据此决定是否调用、怎么填参数
TOOLS = [
    {
        "type": "function",
        "function": {
            "name": "search",
            "description": "网页搜索引擎。当需要回答关于时事、事实以及知识库中找不到的信息时使用。",
            "parameters": {
                "type": "object",
                "properties": {
                    "query": {"type": "string", "description": "搜索关键词"},
                },
                "required": ["query"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "get_current_date",
            "description": "获取今天的日期，格式为 YYYY-MM-DD。",
            "parameters": {"type": "object", "properties": {}},
        },
    },
]

# 工具名到实际函数的映射，模型只负责"决定调用"，执行由我们完成
TOOL_FUNCTIONS: dict[str, Callable[..., str]] = {
    "search": search,
    "get_current_date": get_current_date,
}


class FunctionCallAgent:
    def __init__(self, llm_client: LLM, max_steps: int = 5):
        self.llm_client = llm_client
        self.max_steps = max_steps

    def run(self, question: str) -> str | None:
        messages: list = [
            {"role": "system", "content": "你是一个可以调用工具的智能助手。"},
            {"role": "user", "content": question},
        ]

        for current_step in range(1, self.max_steps + 1):
            print(f"--- 第 {current_step} 步 ---")

            # 1. 把对话和工具定义一起发给模型
            response = self.llm_client.client.chat.completions.create(
                model=self.llm_client.model,
                messages=messages,
                tools=TOOLS,
            )
            message = response.choices[0].message

            # 2. 没有工具调用，说明模型已经给出最终答案
            if not message.tool_calls:
                print(f"最终答案: {message.content}")
                return message.content

            # 3. 先把模型这条带 tool_calls 的回复加入对话
            messages.append(message)

            # 4. 逐个执行工具（模型可能一次请求多个），结果以 role="tool" 返回
            for tool_call in message.tool_calls:
                name = tool_call.function.name
                arguments = json.loads(tool_call.function.arguments or "{}")
                print(f"Action: {name}({arguments})")

                tool_function = TOOL_FUNCTIONS.get(name)
                if tool_function:
                    observation = tool_function(**arguments)
                else:
                    observation = f"错误:未找到名为 '{name}' 的工具。"
                print(f"Observation: {observation}")

                messages.append(
                    {
                        "role": "tool",
                        "tool_call_id": tool_call.id,
                        "content": observation,
                    }
                )

        print("已达到最大步数，流程终止。")
        return None


if __name__ == "__main__":
    agent = FunctionCallAgent(LLM())
    agent.run("英伟达最新的GPU型号是什么？")
