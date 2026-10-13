from typing import Optional


class HelloAgentsLLM:
    def __init__(
        self,
        model_name: Optional[str] = None,
        base_url: Optional[str] = None,
        api_key: Optional[str] = None,
        temperature: Optional[float] = None,
        max_tokens: Optional[int] = None,
        timeout: Optional[int] = None,
        provider: Optional[str] = None,
        budget: Optional[float] = None,
        **kwargs,
    ):
        pass

    def greet(self, name: str) -> str:
        return f"Hello, {name}! Welcome to the Agents LLM."
