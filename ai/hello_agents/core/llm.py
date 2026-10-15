import os
import math

from typing import Optional

from .budget import RunBudget, budgeted


class HelloAgentsLLM:
    def __init__(
        self,
        model: Optional[str] = None,
        base_url: Optional[str] = None,
        api_key: Optional[str] = None,
        temperature: Optional[float] = None,
        max_tokens: Optional[int] = None,
        timeout: Optional[int] = None,
        provider: Optional[str] = None,
        budget: Optional[float] = None,
        **kwargs,
    ):
        self.model = model or os.getenv("LLM_MODEL_ID")
        self.base_url = base_url or os.getenv("LLM_BASE_URL")
        self.api_key = api_key or os.getenv("LLM_API_KEY")
        self.timeout = (
            timeout if timeout is not None else float(os.getenv("LLM_TIMEOUT", "60"))
        )

        if (
            isinstance(self.timeout, bool)
            or not isinstance(self.timeout, (int, float))
            or not math.isfinite(self.timeout)
            or self.timeout <= 0
        ):
            raise ValueError("timeout must be finite and positive")

        if budget is not None and not isinstance(budget, RunBudget):
            raise ValueError("budget must be RunBudget")
        self.budget = budget

        self.temperature = temperature
        self.max_tokens = max_tokens

        if {"messages", "tools", "stream", "model", "tool_choice"} & kwargs.keys():
            raise ValueError(
                "messages/tools/stream/model/tool_choice belong to call methods"
            )

    def greet(self, name: str) -> str:
        return f"Hello, {name}! Welcome to the Agents LLM."
