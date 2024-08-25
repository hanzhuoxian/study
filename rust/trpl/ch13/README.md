## 闭包定义

```rust
fn add_one(num: i32) -> i32 {num+1}

let add_one_v1 = |num: i32| -> i32 {num+1};
```

## 捕获引用或者移动所有权

不可变借用
```rust
    let list = vec![1,2,3];
    let only_borrows = || println!("From closure: {:?}", list);
    only_borrows();
    println!("From outside: {:?}", list);
```

可变借用
```rust

    let mut list = vec![1,2,3];
    let mut borrows_mutably = || list.push(7);
    borrows_mutably();
    println!("From borrows_mutably: {:?}", list);
```

移动所有权

```rust
    let list = vec![1,2,3];

    thread::spawn(move ||println!("From thread: {:?}", list))
    .join()
    .unwrap();
```

## 将被捕获的值移出闭包和Fn trait

FnOnce: 一个会将捕获的值从闭包体中移出的闭包只会实现 FnOnce

```rust
    let list = vec![1,2,3];
    let fn_once = || {list};
    println!("From mut_fn: {:?}", fn_once());
```

FnMut: 适用于不会将捕获的值移出闭包体，但可能会修改捕获值的闭包。这类闭包可以被调用多次。

```rust
    let mut list = vec![1,2,3];
    let mut borrows_mutably = || list.push(7);
    // println!("From borrows_mutably: {:?}", list); // error
    borrows_mutably();
    println!("From borrows_mutably: {:?}", list);
```

Fn: 适用于既不将捕获的值移出闭包体，也不修改捕获值的闭包，同时也包括不从环境中捕获任何值的闭包。

```rust
    let list = vec![1,2,3];
    let only_borrows = || println!("From closure: {:?}", list);
    only_borrows();
    println!("From outside: {:?}", list);
```
