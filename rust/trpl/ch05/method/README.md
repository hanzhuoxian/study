## 方法语法

方法（method）与函数类似：它们使用 `fn` 关键字和名称声明，可以拥有参数和返回值，同时包含在某处调用该方法时会执行的代码。
不过方法与函数是不同的，因为它们在结构体的上下文中被定义（或者是枚举或 `trait` 对象的上下文，将分别在第六章和第十七章讲解），
并且它们第一个参数总是 self，它代表调用该方法的结构体实例。


## 关联函数

所有在 impl 块中定义的函数被称为 关联函数(associated functions)，因为它们与 impl 后
面命名的类型相关。我们可以定义不以 self 为第一参数的关联函数(因此不是方法)，因为 它们并不作用于一个结构体的实例。
我们已经使用了一个这样的函数:在 String 类型上定义 的 String::from 函数。

```rust

impl Rectangle {

    fn square(size: u32) -> Self { // 不以 self 开头，关键字 Self 在函数的返回类型中代指在 impl 关键字后出现的类型
        Self {
            width: size,
            height: size,
        }
    }

}

Rectangle::square(5); // 使用结构体名和:: 语法来调用这个关联函数
```

## 多个 impl 块

我们可以在多个 impl 块中实现方法，这样我们可以将相关的方法分组放在一起。这是一个常见的模式，因为这样可以使代码更易于阅读。

```rust
impl Rectangle {
    fn area(&self) -> u32 {
        self.width * self.height
    }
}

impl Rectangle {
    fn square(size: u32) -> Self {
        Self {
            width: size,
            height: size,
        }
    }
}
```
