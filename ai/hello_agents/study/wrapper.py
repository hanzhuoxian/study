# 装饰器
def my_decorator(func):
    def wrapper():
        print("开始执行")
        func()
        print("结束执行")

    return wrapper


def hello():
    print("Hello!")


hello = my_decorator(hello)
hello()


# @ 是上面的语法糖
@my_decorator
def hi():
    print("hi")


hi()


class Math:
    @staticmethod
    def add(a, b):
        return a + b


print(Math.add(1, 2))


class User:
    def __init__(self, name, age):
        self.name = name
        self.age = age

    @classmethod
    def from_birth_year(cls, name, year):
        return cls(name, 2024 - year)


u = User.from_birth_year("Alice", 1990)
print(u)


class Temperature:
    def __init__(self, celsius):
        self._celsius = celsius

    @property
    def celsius(self):
        return self._celsius

    @celsius.setter
    def celsius(self, value):
        if value < -273.15:
            raise ValueError("Temperature cannot be below absolute zero")
        self._celsius = value

    @property
    def fahrenheit(self):
        return (self._celsius * 9 / 5) + 32


t = Temperature(25)
print(t.celsius)  # 输出: 25
print(t.fahrenheit)  # 输出: 77.0
# t.celsius = -300


class CountCalls:
    def __init__(self, func):
        self.func = func
        self.count = 0

    def __call__(self, *args, **kwargs):
        self.count += 1
        print(f"Function {self.func.__name__} has been called {self.count} times")
        return self.func(*args, **kwargs)


@CountCalls
def say_hello():
    print("Hello!")


say_hello()
say_hello()
