import json
import re

from llm.llm import LLM
from search.search import search
from tools.tools import ToolExecutor

REACT_PROMPT_TEMPLATE = """
你是一个可以调用外部工具的智能助手，按照 ReAct 方式（思考 → 行动 → 观察）一步步解决问题。

## 可用工具
{tools}

## 输出格式
每次只输出**一个** JSON 对象，不要输出任何其他文字，也不要用 ```json 代码块包裹。

调用工具时:
{{"thought": "你的分析和下一步计划", "action": "call_tool", "tool_name": "工具名", "tool_input": "传给工具的参数"}}

给出最终答案时:
{{"thought": "为什么现在可以回答了", "action": "finish", "answer": "最终答案"}}

## 规则
1. tool_name 必须是上面列出的工具之一，名字要完全一致。
2. 每次只调用一个工具，然后等待返回结果（Observation），不要自己编造结果。
3. 信息足够回答问题时，立刻使用 finish，不要重复搜索。
4. 如果工具返回的结果没有帮助，换一个关键词，或者直接说明无法回答。

## 任务
Question: {question}

## 历史记录
{history}
"""


class ReActAgent:
    def __init__(
        self, llm_client: LLM, tool_executor: ToolExecutor, max_steps: int = 5
    ):
        self.llm_client = llm_client
        self.tool_executor = tool_executor
        self.max_steps = max_steps
        self.history: list[str] = []

    def run(self, question: str) -> str | None:
        """
        运行ReAct智能体来回答一个问题。
        """
        self.history = []  # 每次运行时重置历史记录

        for current_step in range(1, self.max_steps + 1):
            print(f"--- 第 {current_step} 步 ---")

            # 1. 格式化提示词
            prompt = REACT_PROMPT_TEMPLATE.format(
                tools=self.tool_executor.getAvailableTools(),
                question=question,
                history="\n".join(self.history) or "无",
            )

            # 2. 调用LLM进行思考
            response_text = self.llm_client.generate(prompt)["text"]
            if not response_text:
                print("错误:LLM未能返回有效响应。")
                return None

            # 3. 解析LLM输出
            try:
                step = self._parse_output(response_text)
            except ValueError as e:
                print(f"解析失败: {e}")
                self.history.append(
                    "Observation: 输出格式错误，请只返回一个合法的 JSON 对象。"
                )
                continue
            print(f"Thought: {step.get('thought')}")

            # 4. 结束或执行工具
            if step.get("action") == "finish":
                answer = step.get("answer")
                print(f"最终答案: {answer}")
                return answer

            tool_name = step.get("tool_name")
            tool_input = step.get("tool_input")
            print(f"Action: {tool_name}[{tool_input}]")
            tool_function = self.tool_executor.getTool(tool_name)
            if tool_function:
                observation = tool_function(tool_input)
            else:
                observation = f"错误:未找到名为 '{tool_name}' 的工具。"
            print(f"Observation: {observation}")

            # 5. 整合到历史记录
            self.history.append(json.dumps(step, ensure_ascii=False))
            self.history.append(f"Observation: {observation}")

        print("已达到最大步数，流程终止。")
        return None

    @staticmethod
    def _parse_output(text: str) -> dict:
        """解析模型输出的 JSON，兼容 ```json 代码块和前后多余的文字。"""
        match = re.search(r"\{.*\}", text, re.DOTALL)
        if not match:
            raise ValueError(f"模型输出中没有找到 JSON: {text}")
        try:
            return json.loads(match.group())
        except json.JSONDecodeError as e:
            raise ValueError(f"JSON 不合法: {e}") from e


if __name__ == "__main__":
    tool_executor = ToolExecutor()
    tool_executor.registerTool(
        "Search",
        "一个网页搜索引擎。当你需要回答关于时事、事实以及在你的知识库中找不到的信息时，应使用此工具。",
        search,
    )
    agent = ReActAgent(llm_client=LLM(), tool_executor=tool_executor)
    agent.run("英伟达最新的GPU型号是什么？")
