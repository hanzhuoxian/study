"""LLM 响应对象定义"""

from dataclasses import dataclass, field
from typing import Dict, List, Optional, Any


# 自动生成初始化、相等比较等方法；手动定义的 __str__、__repr__ 会保留。
@dataclass
class ToolCall:
    """保存一次工具调用的信息，本身不执行工具。"""

    # 类型标注不会在运行时自动检查类型。
    id : str
    name: str
    # 通常是 JSON 字符串，需通过 json.loads() 解析后才能按字典访问。
    arguments: str

    def __str__(self) -> str:
        """print(obj)、str(obj) 时调用，返回易读的调用信息。"""
        return f"ToolCall(id={self.id}, name={self.name}, arguments={self.arguments})"

    def __repr__(self) -> str:
        """repr(obj) 或在列表中显示对象时调用，用于调试。"""
        return "repr:" + self.__str__()

@dataclass
class LLMToolResponse:
    """包含工具调用的响应；模型可能只请求调用工具而没有文本正文。"""

    content: Optional[str]
    # 模型在本次响应中请求的工具调用列表。
    tool_call: List[ToolCall]
    model: str
    usage: Dict[str, Any] = field(default_factory=dict)
    latency_ms: int = 0
    finish_reason: Optional[str] = None
    reasoning_content: Optional[str] = None

@dataclass
class LLMResponse:
    """普通模型响应，保存正文、模型名称和调用统计信息。"""

    content: str
    model: str
    # default_factory 为每个实例创建独立字典，保存 token 用量等统计信息。
    usage: Dict[str, Any] = field(default_factory=dict)
    # 调用耗时，单位为毫秒。
    latency_ms: int = 0
    # 生成结束的原因，如 stop、length；具体取值由模型接口决定。
    finish_reason: Optional[str] = None
    # 模型接口提供的推理内容，没有时为 None。
    reasoning_content: Optional[str] = None

    def __str__(self) -> str:
        """print(response)、str(response) 时直接显示正文。"""
        return self.content

    def __repr__(self) -> str:
        """返回调试摘要，展示耗时、token 数等信息，避免输出完整正文。"""
        parts = [
            f"LLMResponse(model={self.model}",
            f"latency={self.latency_ms}ms",
            f"tokens={self.usage.get('total_tokens', 0)}",
        ]
        if self.reasoning_content:
            parts.append(f"has_reasoning=True")
        parts.append(f"content_length={len(self.content)})")
        return ", ".join(parts)

    def to_dict(self) -> Dict[str, Any]:
        """普通实例方法：将字段整理为字典，便于后续序列化。"""
        return {
            "content": self.content,
            "model": self.model,
            "usage": self.usage,
            "latency_ms": self.latency_ms,
            "finish_reason": self.finish_reason,
            "reasoning_content": self.reasoning_content,
        }

@dataclass
class StreamStats:
    """保存流式响应的统计信息，不保存流式输出的正文。"""

    model: str
    usage: Dict[str, Any] = field(default_factory=dict)
    latency_ms: int = 0
    finish_reason: Optional[str] = None
    reasoning_content: Optional[str] = None
    def to_dict(self) -> Dict:
        """转换为字典，仅在推理内容非空时加入 reasoning_content 字段。"""
        result = {
            "model": self.model,
            "usage": self.usage,
            "latency_ms": self.latency_ms,
            "finish_reason": self.finish_reason,
        }
        if self.reasoning_content:
            result["reasoning_content"] = self.reasoning_content
        return result
