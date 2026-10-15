"""管理模型调用的次数、token 用量和耗时预算，提供普通及流式调用装饰器。"""

import math
from time import monotonic
from threading import Lock
from contextvars import ContextVar
from contextlib import contextmanager
from functools import wraps

# 保存当前执行上下文的预算；嵌套作用域退出后恢复外层预算。
_current = ContextVar('helloagents_budget', default=None)

class BudgetExceeded(RuntimeError):
    """预算耗尽或无法确认 token 用量时抛出的异常。"""

    pass


class RunBudget:
    """跟踪一组模型调用共享的预算。

    max_calls 和 max_tokens 可为零，空值表示不限制该项；max_seconds
    必须为有限的正数。耗时从实例创建时开始累计，进入预算作用域 不会重新计时。
    token 在调用结束后累计，单次调用可能使总量超过上限；后续调用开始前
    才会检查累计用量，此处不会预估本次调用的 token 消耗。
    """

    def __init__(
        self,
        *,
        max_calls: int | None = None,
        max_tokens: int | None = None,
        max_seconds: float | None = None,
    ):
        # 精确判断整数类型，避免将布尔值当作预算值接受。
        for value in (max_calls, max_tokens):
            if value is not None and (type(value) is not int or value < 0):
                raise ValueError("Budget values must be non-negative.")
        if max_seconds is not None and (
            isinstance(max_seconds, bool)
            or not isinstance(max_seconds, (int, float))
            or not math.isfinite(max_seconds)
            or max_seconds <= 0
        ):
            raise ValueError("max_seconds must be finite and positive")
        self.max_calls = max_calls
        self.max_tokens = max_tokens
        self.max_seconds = max_seconds
        self.calls = 0
        self.tokens = 0
        self.unknown_usage = 0
        # 单调时钟不受系统时间调整影响，锁保护计数的检查与更新。
        self._started = monotonic()
        self._lock = Lock()

    @contextmanager
    def scope(self):
        """在代码块内启用此预算，正常或异常退出时均恢复原上下文。

        支持嵌套作用域；进入上下文时返回此实例，便于在代码块内访问。
        """
        token = _current.set(self)
        try:
            yield self
        finally:
            _current.reset(token)

    def check_time(self):
        """检查累计耗时；仅在严格超过 max_seconds 时抛出 BudgetExceeded。"""
        if self.max_seconds is not None:
            elapsed = monotonic() - self._started
            if elapsed > self.max_seconds:
                raise BudgetExceeded(
                    f"Time budget exceeded: {elapsed:.2f}s > {self.max_seconds:.2f}s"
                )

    def reserve(self):
        """调用前检查各项预算，并在锁内占用一次调用额度。

        已占用的次数不会因后续调用失败而退还。设置 token 上限时，先前
        存在未知用量也会阻止继续调用，以免无法判断剩余额度。
        """
        with self._lock:
            self.check_time()
            if self.max_calls is not None and self.calls >= self.max_calls:
                raise BudgetExceeded('Model call budget exhausted')
            if self.max_tokens is not None and (self.unknown_usage or self.tokens >= self.max_tokens):
                raise BudgetExceeded('Token budget exhausted or previous usage unknown')
            self.calls += 1

    def record(self, usage):
        """累加本次调用报告的 token 用量，或将其记为用量未知。

        usage 应为字典，优先读取 total_tokens；仅在该字段缺失或为 None
        时尝试将 prompt_tokens 与 completion_tokens 相加。有效计数必须
        为非负整数（不接受布尔值）；缺失或无效的统计不会按零用量处理。
        """
        total = usage.get('total_tokens') if isinstance(usage, dict) else None
        if total is None and isinstance(usage, dict):
            parts = [usage.get('prompt_tokens'), usage.get('completion_tokens')]
            if all(type(x) is int and x >= 0 for x in parts):
                total = sum(parts)
        with self._lock:
            if type(total) is not int or total < 0:
                self.unknown_usage += 1
            else:
                self.tokens += total

    def snapshot(self):
        """在锁内读取调用次数、已报告 token、未知用量次数及累计耗时。"""
        with self._lock:
            return dict(model_calls=self.calls, reported_tokens=self.tokens,
                        unknown_usage_calls=self.unknown_usage,
                        elapsed_seconds=monotonic() - self._started)


@contextmanager
def budget_call(Default=None):
    """为一次调用预留额度，并通过可变 receipt 收集用量。

    当前作用域的预算优先于默认预算；两者都为空时不执行预算检查。
    返回 (receipt, budget)，调用方将统计写入 receipt['usage']。
    正常退出时检查耗时；只要选中了预算，退出时就将用量记录到该预算。
    调用失败时也会记录，未填写的用量会记为未知。
    """
    budget = _current.get() or Default
    receipt = {}
    if budget is not None:
        budget.reserve()

    try:
        yield receipt,budget
        if budget is not None:
            budget.check_time()
    finally:
        if budget is not None:
            budget.record(receipt.get('usage', {}))

def budgeted(method):
    """包装普通调用；宿主需提供 budget，返回结果需提供 usage 属性。"""
    @wraps(method)
    def call(self, *args, **kwargs):
        with budget_call(self.budget) as (receipt, budget):
            result = method(self, *args, **kwargs)
            receipt['usage'] = result.usage
            return result
    return call

def budgeted_stream(method):
    """包装同步流式调用，每次产出前检查耗时，退出时关闭源流。

    开始迭代时才预留额度；仅在源流自然结束后读取 last_call_stats.usage，
    提前关闭或异常退出时不会写入该统计。源流需支持 close()。
    """
    @wraps(method)
    def call(self, *args, **kwargs):
        with budget_call(self.budget) as (receipt, budget):
            # 清除上一轮统计，避免复用过期的用量。
            self.last_call_stats = None
            source = method(self, *args, **kwargs)
            try:
                for item in source:
                    if budget:
                        budget.check_time()
                    yield item
                receipt['usage'] = getattr(self.last_call_stats, 'usage', None)
            finally:
                source.close()
    return call


def budgeted_astream(method):
    """包装异步流式调用，每次产出前检查耗时，退出时异步关闭源流。

    开始迭代时才预留额度，自然结束后才读取 last_call_stats.usage；
    宿主需提供 budget 和 last_call_stats，源流需支持 aclose()。
    """
    @wraps(method)
    async def call(self, *args, **kwargs):
        with budget_call(self.budget) as (receipt, budget):
            # 清除上一轮统计，避免复用过期的用量。
            self.last_call_stats = None
            source = method(self, *args, **kwargs)
            try:
                async for item in source:
                    if budget:
                        budget.check_time()
                    yield item
                receipt['usage'] = getattr(self.last_call_stats, 'usage', None)
            finally:
                await source.aclose()
    return call
    