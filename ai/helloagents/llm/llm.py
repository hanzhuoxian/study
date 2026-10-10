import os

from dotenv import load_dotenv
from openai import OpenAI, OpenAIError

load_dotenv()


class LLM:
    def __init__(
        self,
        model: str | None = None,
        api_key: str | None = None,
        base_url: str | None = None,
        timeout: int | None = None,
    ):
        self.model = model or os.getenv("LLM_MODEL_ID")
        self.api_key = api_key or os.getenv("LLM_API_KEY")
        self.base_url = base_url or os.getenv("LLM_BASE_URL")
        self.timeout = timeout or 180  # default timeout in seconds

        if not self.api_key:
            raise ValueError("API key is required for LLM initialization.")

        self.client = OpenAI(
            api_key=self.api_key, base_url=self.base_url, timeout=self.timeout
        )

    def generate(
        self,
        prompt: str | list[dict],
        stream: bool = False,
        max_tokens: int | None = None,
        temperature: float = 0,
    ) -> dict:
        """
        Generate text based on the provided prompt.

        :param prompt: The input prompt, a string or a list of chat messages.
        :param stream: Whether to stream the response.
        :param max_tokens: The maximum number of tokens to generate.
            None means the provider's default limit is used.
        :param temperature: Sampling temperature for randomness in generation.
        :return: A dictionary containing the generated text and other metadata.
        """
        if isinstance(prompt, str):
            messages = [{"role": "user", "content": prompt}]
        else:
            messages = prompt
        kwargs = {"model": self.model, "messages": messages, "temperature": temperature}
        if max_tokens is not None:
            kwargs["max_tokens"] = max_tokens
        try:
            if stream:
                return self._stream_response(kwargs)
            else:
                return self._get_response(kwargs)
        except OpenAIError as e:
            raise RuntimeError(f"Error during text generation: {e}") from e

    def _stream_response(self, kwargs: dict) -> dict:
        """
        Stream the response from the LLM.
        """
        response = self.client.chat.completions.create(**kwargs, stream=True)

        generated_text = ""
        finish_reason = None
        for chunk in response:
            if not chunk.choices:
                continue
            choice = chunk.choices[0]
            if choice.delta.content:
                generated_text += choice.delta.content
            finish_reason = choice.finish_reason or finish_reason

        self._warn_if_truncated(finish_reason)
        return {"text": generated_text}

    def _get_response(self, kwargs: dict) -> dict:
        """
        Get the response from the LLM without streaming.
        """
        response = self.client.chat.completions.create(**kwargs)

        choice = response.choices[0]
        self._warn_if_truncated(choice.finish_reason)
        return {"text": choice.message.content}

    @staticmethod
    def _warn_if_truncated(finish_reason: str | None) -> None:
        if finish_reason == "length":
            print("⚠️ 输出达到 max_tokens 上限，内容被截断")


# --- 客户端使用示例 ---
if __name__ == "__main__":
    try:
        llm_client = LLM()

        example_messages = [
            {
                "role": "system",
                "content": "You are a helpful assistant that writes Python code.",
            },
            {"role": "user", "content": "写一个快速排序算法"},
        ]

        print("--- 调用LLM ---")
        response_text = llm_client.generate(prompt=example_messages)
        if response_text:
            print("\n\n--- 完整模型响应 ---")
            print(response_text)

    except ValueError as e:
        print(e)
