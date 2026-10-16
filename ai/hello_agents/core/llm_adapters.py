from abc import ABC, abstractmethod
import asyncio
import time
import threading
import queue as thread_queue

from urllib.parse import urlparse
from typing import Any, Dict, Iterable, List, Optional

from core.llm_response import LLMResponse, LLMToolResponse


def _openai_usage(value) -> Dict[str, Any]:
    """保留服务端报告的 Token 明细，不补造缺失的缓存指标。"""
    if value is None:
        return {}
    if isinstance(value, dict):
        return dict(value)
    if callable(getattr(value, "model_dump", None)):
        return value.model_dump(exclude_none=True)
    return {
        name: getattr(value, name)
        for name in (
            "prompt_tokens",
            "completion_tokens",
            "total_tokens",
            "prompt_tokens_details",
            "completion_tokens_details",
            "prompt_cache_hit_tokens",
            "prompt_cache_miss_tokens",
        )
        if hasattr(value, name) and getattr(value, name) is not None
    }


class BaseLLMAdapter(ABC):
    def __init__(self, api_key: str, base_url: Optional[str], timeout: int, model: str):
        self.api_key = api_key
        self.base_url = base_url
        self.timeout = timeout
        self.model = model
        self._client = None
        self._async_client = None
        self._client_lock = threading.Lock()

    def _get_client(self):
        with self._client_lock:
            if self._client is None:
                self._client = self.create_client()
            return self._client

    @abstractmethod
    def create_client(self) -> Any:
        """创建客户端实例"""
        pass

    def create_async_client(self) -> Any:
        """创建异步客户端实例（子类可选实现）"""
        return None

    @abstractmethod
    def invoke(self, prompt: str, **kwargs) -> LLMResponse:
        """非流式调用 LLM 模型，返回完整响应"""
        pass

    @abstractmethod
    def stream_invoke(self, prompt: str, **kwargs) -> Iterable[str]:
        """流式调用，返回生成器"""
        pass

    async def astream_invoke(self, messages: List[Dict], **kwargs):
        """使用有界缓冲的回退实现；取消后，在当前阻塞读取结束时停止拉取。"""
        items = thread_queue.Queue(maxsize=8)
        stopped = threading.Event()
        end = object()

        def put(value):
            while not stopped.is_set():
                try:
                    items.put(value, timeout=0.05)
                    return True
                except thread_queue.Full:
                    continue
            return False

        def produce():
            source = self.stream_invoke(messages, **kwargs)
            try:
                while not stopped.is_set():
                    try:
                        item = next(source)
                    except StopIteration:
                        break
                    if not put(item):
                        break
            except Exception as e:
                put(e)
            finally:
                close = getattr(source, "close", None)
                if callable(close):
                    close()
                put(end)

        worker = threading.Thread(
            target=produce, daemon=True, name="helloagents-stream"
        )
        worker.start()

        try:
            while True:
                try:
                    item = items.get_nowait()
                except thread_queue.Empty:
                    await asyncio.sleep(0.01)
                    continue
                if item is end:
                    break
                if isinstance(item, Exception):
                    raise item
                yield item
        finally:
            stopped.set()

    def close(self):
        if self._async_client is not None:
            try:
                asyncio.get_running_loop()
            except RuntimeError:
                asyncio.run(self.aclose)
                return
            raise RuntimeError("Use await llm.aclose() for async clients")

        if self._client is not None:
            self._client.close()
            self._client = None

    async def aclose(self):
        if self._async_client is not None:
            await self._async_client.close()
            self._async_client = None
        if self._client is not None:
            await asyncio.to_thread(self._client.close)
            self._client = None

    @abstractmethod
    def invoke_with_tools(
        self, messages: List[Dict], tools: List[Dict], **kwargs
    ) -> LLMToolResponse:
        """工具调用（Function Calling）"""
        pass

    def _is_thinking_model(self, model_name: str) -> bool:
        """判断是否为thinking model"""
        thinking_keywords = ["reasoner", "o1", "o3", "thinking"]
        model_lower = model_name.lower()
        return any(keyword in model_lower for keyword in thinking_keywords)

    async def astream_invoke_with_tools(self, messages, tools, **kwargs):
        """回退适配器只产出一次完整响应，不模拟逐 Token 流式输出。"""
        if tools:
            response = await asyncio.to_thread(
                self.invoke_with_tools, messages, tools, **kwargs
            )
        else:
            kwargs.pop("tool_choice", None)
            response = await asyncio.to_thread(self.invoke, messages, **kwargs)
        if response.content:
            yield response.content
        yield response


class OpenAIAdapter(BaseLLMAdapter):
    def create_client(self) -> Any:
        from openai import OpenAI

        return OpenAI(
            api_key=self.api_key, base_url=self.base_url, timeout=self.timeout
        )

    def create_async_client(self) -> Any:
        """创建OpenAI异步客户端"""
        from openai import AsyncOpenAI

        return AsyncOpenAI(
            api_key=self.api_key, base_url=self.base_url, timeout=self.timeout
        )

    def invoke(self, messages: List[Dict], **kwargs) -> LLMResponse:
        self._get_client()
        start_time = time.time()
        try:
            response = self._client.chat.completions.create(
                model=self.model, messages=messages, **kwargs
            )
            latency_ms = int((time.time() - start_time) * 1000)
            choice = response.choices[0]
            content = choice.message.get("content", "")
            reasoning_content = None

            reasoning_content = choice.message.get("reasoning_content", None)

            usage = {}
            if hasattr(response, "usage") and response.usage:
                usage = _openai_usage(response.usage)
        except Exception as e:
            raise e


class AnthropicAdapter(BaseLLMAdapter):
    pass


class GeminiAdapter(BaseLLMAdapter):
    pass


def create_adapter(api_key, base_url, timeout, model, provider=None) -> BaseLLMAdapter:
    """显式指定的协议优先于主机名识别，代理地址也遵循此规则。"""
    adapters = {
        "openai": OpenAIAdapter,
        "anthropic": AnthropicAdapter,
        "gemini": GeminiAdapter,
    }
    aliases = {
        "deepseek": "openai",
        "qwen": "openai",
        "ollama": "openai",
        "claude": "anthropic",
        "google": "gemini",
    }
    if provider is not None:
        if not isinstance(provider, str):
            raise TypeError("provider must be a protocol name")
        provider = aliases.get(provider.lower(), provider.lower())
        if provider not in adapters:
            raise ValueError("provider must be openai, anthropic or gemini")
    else:
        host = (urlparse(base_url or "").hostname or "").lower()
        provider = (
            "anthropic"
            if host == "anthropic.com" or host.endswith(".anthropic.com")
            else (
                "gemini"
                if host == "googleapis.com" or host.endswith(".googleapis.com")
                else "openai"
            )
        )
    return adapters[provider](api_key, base_url, timeout, model)
