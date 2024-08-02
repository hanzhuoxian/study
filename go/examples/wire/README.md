# wire Go 依赖注入

wire 是一个代码生成工具，它使用依赖注入自动连接组件。组件之间的依赖关系在wire 中表示为函数参数，鼓励显式初始化，而不是全局变量。由于 wire 没有运行时状态或反射，因此编写用于 wire 的代码即使对于手写的初始化也很有用。

## 安装

```bash
go get github.com/google/wire/cmd/wire
```

$GOPATH/bin 添加到你的环境变量 $PATH 中

## 文档

[Tutorial](https://github.com/google/wire/blob/master/_tutorial/README.md)  
[User Guide](https://github.com/google/wire/blob/master/docs/guide.md)  
[Best Practices](https://github.com/google/wire/blob/master/docs/best-practices.md)  
[FQA](https://github.com/google/wire/blob/master/docs/faq.md)  

## 最佳实战

### 类型区别

如果你需要一个基本类型来注入，请创建一个新的类型来避免冲突

```go
type MySQLConnectionString string
```

### 结构体选项


-----------我也是有底线的-----------
