use std::borrow::Borrow;
use std::ops::Deref;

fn main() {
    let x = "hello".to_string();
    // 手动解引用 把 &String 转换成 &str 的三种方式

    // 直接调用 deref
    match x.deref() {
        "hello" => println!("hello!"),
        _ => {}
    }

    // String 提供的 as_ref
    match x.as_ref() {
        "hello" => println!("hello!"),
        _ => {}
    }

    // 定义与 borrow 与 AsRef trait 功能一致
    match x.borrow() {
        "hello" => println!("hello!"),
        _ => {}
    }

    // 使用 *将 String 转换为 str，然后再用引用操作符转为 &str
    match &*x {
        "hello" => println!("hello!"),
        _ => {}
    }

    // 因为 String 类型的 index 操作可以返回 &str
    match &x[..] {
        "hello" => println!("hello!"),
        _ => {}
    }

}
