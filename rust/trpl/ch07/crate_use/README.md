## 使用 use 将模块引入作用域

```rust
mod front_of_house {
    pub mod hosting {
        pub fn add_to_waitlist() {}
    }
}

use crate::front_of_house::hosting;

pub fn eat_at_restaurant() {
    hosting::add_to_waitlist();
}

```

## use 只能创建特定作用域的短路径

```rust
mod front_of_house {
    pub mod hosting {
        pub fn add_to_waitlist() {}
    }
}

use crate::front_of_house::hosting;

mod customer {
    use crate::front_of_house::hosting; // 如果没有这个将不能编译，上面的 use 不生效

    pub fn eat_at_restrauant() {
        hosting::add_to_waitlist();
    }
}

```

## 创建惯用的 use 路径

use 引入函数只引入函数的父模块引入作用域，而不是函数本身

use 引入结构体和枚举时，引入完整路径。

```rust
use std::collections::HashMap;
let mut map = HashMap::new();
map.insert(1, 2);
```

使用父模块引入同名但是不同父模块的 Result

```rust
use std::fmt;
use std::io;

fn function1() -> fmt::Result {
    // --snip--
}

fn function2() -> io::Result<()> {
    // --snip--
}
```

使用 as 关键字提供新名称


```rust
use std::fmt::Result;
use std::io::Result as IoResult;

fn function1() -> Result {
    // --snip--
}

fn function2() -> IoResult<()> {
    // --snip--
}
```

## 使用 pub use 重导出名称（re-exporting）

```rust
mod front_of_house {
    pub mod hosting {
        pub fn add_to_waitlist() {}
    }
}

pub use crate::front_of_house::hosting;

pub fn eat_at_restaurant() {
    hosting::add_to_waitlist();
}
```

## 使用外部包

标准包是外部包不需要修改 cargo.toml 但是也需要 use
```rust
use rand::Rng;
let secret_number = rand::thread_rng().gen_range(1..=100);

```

## 使用嵌套路径来消除大量 use 行

```rust
use std::cmp::Ordering;
use std::io;
// 等价于
use std::{cmp::Ordering, io}; // 嵌套路径

use std::io;
use std::io::Write;
// 等价于
use std::io::{self, Write};
```

## 通过 glob 运算符将所有公有定义引入作用域

- 一般只用于测试

```rust
use std::collections::*;
```
