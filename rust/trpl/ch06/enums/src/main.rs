enum IpAddrKind {
    V4,
    V6,
}
struct IpAddrStruct {
    kind: IpAddrKind,
    address: String,
}

enum IpAddr {
    V4(String),
    V6(String),
}

#[derive(Debug)]
enum IpAddrU8 {
    V4(u8, u8, u8, u8),
    V6(String),
}

#[derive(Debug)]
enum Message {
    Quit,                       // 没有包含任何数据
    Move { x: i32, y: i32 },    // 类似结构体包含命名字段
    Write(String),              //包含单独一个String
    ChangeColor(i32, i32, i32), //包含三个 i32
}

impl Message {
    fn call(&self) {}
}

fn main() {
    let message = Message::Write(String::from("hello"));
    message.call();
    let four = IpAddrKind::V4;
    let six: IpAddrKind = IpAddrKind::V6;
    route(four);
    route(six);

    let home = IpAddrStruct {
        kind: IpAddrKind::V4,
        address: String::from("127.0.0.1"),
    };

    let loopback = IpAddrStruct {
        kind: IpAddrKind::V6,
        address: String::from("::1"),
    };

    let home = IpAddr::V4(String::from("127.0.0.1"));
    let loopback = IpAddr::V6(String::from("::1"));

    let home = IpAddrU8::V4(127, 0, 0, 1);
    let loopback = IpAddrU8::V6(String::from("::1"));
    println!("{:?} {:?}", home, loopback);

    // Option
    let some_number = Some(3);
    let some_char = Some('e');
    let absent_number: Option<i32> = None;

    let some_number = Option::Some(3);
    let some_char = Option::Some('e');
    let absent_number: Option<i32> = Option::None;
}

fn route(ip_kind: IpAddrKind) {}
