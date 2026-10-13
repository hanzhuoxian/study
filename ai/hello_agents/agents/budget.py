import math
from time import monotonic
from threading import Lock
from contextvars import ContextVar
from contextlib import contextmanager

_current = ContextVar('helloagents_budget', default=None)

class BudgetExceeded(RuntimeError):
    """Raised when the budget is exceeded."""

    pass


class RunBudget:
    """Class to manage and track the budget for running agents."""

    def __init__(
        self,
        *,
        max_calls: int | None = None,
        max_tokens: int | None = None,
        max_seconds: float | None = None,
    ):
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
        self.calls_made = 0
        self.tokens_used = 0
        self._started = monotonic()
        self._lock = Lock()

    @contextmanager
    def scope(self):
        """Context manager to set the current budget for the duration of a block."""
        token = _current.set(self)
        try:
            yield self
        finally:
            _current.reset(token)

    def check_time(self):
        """Check if the time budget has been exceeded."""
        if self.max_seconds is not None:
            elapsed = monotonic() - self._started
            if elapsed > self.max_seconds:
                raise BudgetExceeded(
                    f"Time budget exceeded: {elapsed:.2f}s > {self.max_seconds:.2f}s"
                )

    def reserve(self):
        """Reserve a call and check if the budget has been exceeded."""
        with self._lock:
            self.check_time()
            if self.max_calls is not None and self.calls >= self.max_calls:
                raise BudgetExceeded('Model call budget exhausted')
            if self.max_tokens is not None and (self.unknown_usage or self.tokens >= self.max_tokens):
                raise BudgetExceeded('Token budget exhausted or previous usage unknown')
            self.calls += 1

    def record(self, usage):
        """Record the usage of tokens from a model call."""
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
        """Return a snapshot of the current budget usage."""
        with self._lock:
            return dict(model_calls=self.calls, reported_tokens=self.tokens,
                        unknown_usage_calls=self.unknown_usage,
                        elapsed_seconds=monotonic() - self._started)


@contextmanager
def budget_call(Default=None):
    """Context manager to manage the budget for a model call."""
    budget = _current.get() or Default
    receipt = {}
    if budget is not None:
        budget.reserve()

    try:
        yield receipt,budget
        if budget is not None:
            budget.check_time()
    finally:
        if Default is not None:
            budget.record(receipt.get('usage', {}))