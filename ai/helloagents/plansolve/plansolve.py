import ast
import re

from llm.llm import LLM

PLANNER_PROMPT_TEMPLATE = """
你是一个顶级的AI规划专家。你的任务是将用户提出的复杂问题分解成一个由多个简单步骤组成的行动计划。
请确保计划中的每个步骤都是一个独立的、可执行的子任务，并且严格按照逻辑顺序排列。
你的输出必须是一个Python列表，其中每个元素都是一个描述子任务的字符串。

问题: {question}

请严格按照以下格式输出你的计划,```python与```作为前后缀是必要的:
```python
["步骤1", "步骤2", "步骤3", ...]
```
"""

EXECUTOR_PROMPT_TEMPLATE = """
你是一位顶级的AI执行专家。你的任务是严格按照给定的计划，一步步地解决问题。
你将收到原始问题、完整的计划、以及到目前为止已经完成的步骤和结果。
请你专注于解决“当前步骤”，并仅输出该步骤的最终答案，不要输出任何额外的解释或对话。

# 原始问题:
{question}

# 完整计划:
{plan}

# 历史步骤与结果:
{history}

# 当前步骤:
{current_step}

请仅输出针对“当前步骤”的回答:
"""


class Planner:
    def __init__(self, llm_client: LLM):
        self.llm_client = llm_client

    def plan(self, question: str) -> list[str]:
        """
        根据用户问题生成一个行动计划。
        """
        prompt = PLANNER_PROMPT_TEMPLATE.format(question=question)

        print("--- 正在生成计划 ---")
        response_text = self.llm_client.generate(prompt)["text"] or ""

        print(f"✅ 计划已生成:\n{response_text}")

        # 解析LLM输出的列表字符串
        try:
            # 优先取```python代码块中的内容，没有代码块时退回到第一个[...]
            match = re.search(r"```python\s*(.*?)```", response_text, re.DOTALL)
            if not match:
                match = re.search(r"(\[.*\])", response_text, re.DOTALL)
            if not match:
                raise ValueError("响应中没有找到计划列表")
            # 使用ast.literal_eval来安全地执行字符串，将其转换为Python列表
            plan = ast.literal_eval(match.group(1).strip())
        except (ValueError, SyntaxError) as e:
            print(f"❌ 解析计划时出错: {e}")
            print(f"原始响应: {response_text}")
            return []

        if not isinstance(plan, list):
            print(f"❌ 计划不是列表: {plan!r}")
            return []
        return [str(step) for step in plan]


class Executor:
    def __init__(self, llm_client: LLM):
        self.llm_client = llm_client

    def execute(self, question: str, plan: list[str]) -> str:
        """
        根据计划，逐步执行并解决问题。
        """
        history = ""  # 用于存储历史步骤和结果的字符串
        response_text = ""
        plan_str = "\n".join(f"{i}. {step}" for i, step in enumerate(plan, 1))

        print("\n--- 正在执行计划 ---")

        for i, step in enumerate(plan, 1):
            print(f"\n-> 正在执行步骤 {i}/{len(plan)}: {step}")

            prompt = EXECUTOR_PROMPT_TEMPLATE.format(
                question=question,
                plan=plan_str,
                history=history or "无",  # 如果是第一步，则历史为空
                current_step=step,
            )

            response_text = self.llm_client.generate(prompt)["text"] or ""

            # 更新历史记录，为下一步做准备
            history += f"步骤 {i}: {step}\n结果: {response_text}\n\n"

            print(f"✅ 步骤 {i} 已完成，结果: {response_text}")

        # 循环结束后，最后一步的响应就是最终答案
        return response_text


class PlanAndSolveAgent:
    def __init__(self, llm_client: LLM):
        """
        初始化智能体，同时创建规划器和执行器实例。
        """
        self.llm_client = llm_client
        self.planner = Planner(self.llm_client)
        self.executor = Executor(self.llm_client)

    def run(self, question: str) -> str | None:
        """
        运行智能体的完整流程:先规划，后执行。
        """
        print(f"\n--- 开始处理问题 ---\n问题: {question}")

        # 1. 调用规划器生成计划
        plan = self.planner.plan(question)

        # 检查计划是否成功生成
        if not plan:
            print("\n--- 任务终止 --- \n无法生成有效的行动计划。")
            return None

        # 2. 调用执行器执行计划
        final_answer = self.executor.execute(question, plan)

        print(f"\n--- 任务完成 ---\n最终答案: {final_answer}")
        return final_answer


if __name__ == "__main__":
    # 示例用法
    llm_client = LLM()  # 初始化你的LLM客户端
    agent = PlanAndSolveAgent(llm_client)

    # 测试问题
    test_question = "一个水果店周一卖出了15个苹果。周二卖出的苹果数量是周一的两倍。周三卖出的数量比周二少了5个。请问这三天总共卖出了多少个苹果？"
    agent.run(test_question)
