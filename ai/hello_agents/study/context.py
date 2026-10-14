from contextvars import ContextVar
import contextvars
import asyncio

_current = ContextVar('study_context', default="helloagents")

print(_current.get())  # 输出 'helloagents'

token = _current.set('new_context')
print(_current.get())  # 输出 'new_context'

_current.reset(token)
print(_current.get())  # 输出 'helloagents'


v = ContextVar('v', default=0)
print(v.get())  # 输出 0

v.set(10)
ctx = contextvars.copy_context()
print(ctx.get(v))  # 输出 10

ctx.run(v.set, 20)
print(ctx.get(v))  # 输出 20

def f():
    v.set(30)
    return v.get()

print(v.get())  # 输出 10
print(ctx.run(f))  # 输出 30

user = ContextVar('user', default='guest')

async def handle(name):
    user.set(name)
    await asyncio.sleep(1)
    print(f"Hello, {user.get()}!")

async def main():
    await asyncio.gather(handle('Alice'), handle('Bob'))

asyncio.run(main())