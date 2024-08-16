// 结构体定义
struct User {
    active: bool, // 结构体字段定义
    username: String,
    email: String,
    sign_in_count: u64,
}

fn main() {
    // 创建结构体
    let mut user1 = User {
        active: true,
        username: String::from("li"),
        email: String::from("li"),
        sign_in_count: 1,
    };
    // 使用结构体
    println!("{}", user1.username);
    //  结构体字段赋值
    user1.active = false; // 结构体字段赋值
    println!("{}", user1.active);

    // 初始化简写
    let user1 = build_user(String::from("LISI"), String::from("lisi@rust.com"));
    let user2 = User {
        active: user1.active,
        username: user1.username,
        email: String::from("anthor@rust.com"),
        sign_in_count: user1.sign_in_count,
    };
    println!("{} {}", user2.username, user2.email);

    let user1 = build_user(String::from("LISI"), String::from("lisi@rust.com"));
    // 结构体更新语法
    let user2 = User {
        email: String::from("another@example.com"),
        ..user1 // ..user1 必须放在最后，以指定其余的字段应从 user1 的相应字段中获取其值
    };
    println!("{} {} {}", user2.username, user2.email, user1.active);
    // println!("{}", user1.username); //..user1 与赋值操作类似会转移所有权，所以不能使用 user1.username

    struct Color(i32, i32, i32);
    struct Point(i32, i32, i32);
    let black = Color(0, 0, 0);
    let orign = Point(0, 0, 0);

    println!("{} {}", black.0, orign.0);

    #[derive(Debug)]
    struct AlwaysEqual();
    let subject = AlwaysEqual();
    println!("{:?}", subject);
}

fn build_user(username: String, email: String) -> User {
    User {
        active: true,
        username: username,
        email, // 字段初始化简写语法 field init shorthand
        sign_in_count: 1,
    }
}
