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


## 