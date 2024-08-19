## 模块树

代码

```rust
mod front_and_house {
    mod hosting {
        fn add_to_wait_list() {}
        fn seat_at_table() {}
    }

    mod serving {
        fn take_order() {}
        fn server_order() {}
        fn take_payment() {}
    }
}
```

会生成如下模块树

```

crate
 └── front_of_house
     ├── hosting
     │   ├── add_to_wait_list
     │   └── seat_at_table
     └── serving
         ├── take_order
         ├── serve_order
         └── take_payment
```

## 使用 pub 关键字暴露路径


## 使用 super 开始的相对路径

```rust

fn deliver_order() {}
mod back_of_house {
    fn fix_incorrect_order() {
        cook_order();
        super::deliver_order(); // 使用 super 调用父模块的函数
    }

    fn cook_order() {}
}
```

## 创建共有的结构体和枚举

```rust
mod front_and_house {
    pub struct Breakfast {
        pub toast: String,
        seasonal_fruit: String,
    }

    impl Breakfast {
        pub fn summer(toast: &str) -> Breakfast {
            Breakfast {
                toast: String::from(toast),
                seasonal_fruit: String::from("peaches"),
            }
        }
    }


    pub enum Appetizer {
        Soup,
        Salad,
    }
}


fn eat_at_restaurant() {
    let mut meal = front_and_house::Breakfast::summer("Rye");
    meal.toast = String::from("Wheat");
    println!("I'd like {} toast please", meal.toast);
    // meal.seasonal_fruit = String::from("blueberries"); // 错误，seasonal_fruit 是私有的

    // 枚举公有后枚举成员都公有
    let order1 = back_of_house::Appetizer::Soup;
    let order2 = back_of_house::Appetizer::Salad;
}


```
