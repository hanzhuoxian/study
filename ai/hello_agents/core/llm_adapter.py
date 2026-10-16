from abc import ABC, abstractmethod
import threading
from typing import Any, Dict, Iterable, List, Optional

from core.llm_reponse import LLMResponse, LLMToolResponse

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
        """创建并返回一个同步客户端实例。"""
        pass

    def create_async_client(self) -> Any:
        """创建并返回一个异步客户端实例。"""
        return None

    @abstractmethod
    def invoke(self, prompt: str, **kwargs) -> LLMResponse:
        """调用 LLM 模型并返回结果。"""
        pass
    @abstractmethod
    def stream_invoke(self, prompt: str, **kwargs) -> Iterable[str]:
        """调用 LLM 模型并返回一个可迭代的流式响应。"""
        pass
    @abstractmethod
    def invoke_with_tools(self, messages: List[Dict], tools: List[Dict], **kwargs) -> LLMToolResponse:
        """工具调用（Function Calling）"""
        pass

    def _is_thinking_model(self, model_name: str) -> bool:
        """判断是否为thinking model"""
        thinking_keywords = ["reasoner", "o1", "o3", "thinking"]
        model_lower = model_name.lower()
        return any(keyword in model_lower for keyword in thinking_keywords)