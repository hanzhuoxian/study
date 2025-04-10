fn main() {
    // 变量作用域
    // 变量 s 绑定到一个字符串字面值，变量从声明的地方开始到当前作用域结束
    {
        let s = "hello";
        // s 是有效的
        println!("{s}");
    }
    // s 是无效的
    // println!("{s}")

    // String 类型，这个类型管理被分配到堆上的数据

    let mut s = String::from("hello");
    s.push_str(", world!");
    println!("String s : {s}");

    // 变量与数据的交互方式一：移动
    let x: i32 = 5;
    let y = x;
    println!("x = {x}, y = {y}");

    let s1 = String::from("hello");
    let s2 = s1;
    println!("s2 = {s2}"); // s2 是有效的
                           // println!("s1 = {s1}, s2 = {s2}"); // 编译错误，s1 已经被移动到 s2

    // 变量与数据的交互方式二：克隆
    let s1 = String::from("hello");
    let s2 = s1.clone();
    println!("s1 = {s1}, s2 = {s2}");

    ownership_fn();

    owner_resurn();

    ownership_tuple();
}

fn ownership_fn() {
    // 所有权与函数
    let s = String::from("hello"); // s 进入作用域
    takes_ownership(s); // s 的值移动到函数里 ...
                        // ... 所以到这里不再有效
                        // println!("{s}"); // 编译错误
    let x = 5; // x 进入作用域
    makes_copy(x); // x 应该移动函数里，
                   // 但 i32 是 Copy 的，
                   // 所以在后面可继续使用 x
    println!("x = {x}");
}

fn takes_ownership(some_string: String) {
    // some_string 进入作用域
    println!("{}", some_string);
} // 这里，some_string 移出作用域并调用 `drop` 方法。
  // 占用的内存被释放

fn makes_copy(some_integer: i32) {
    // some_integer 进入作用域
    println!("{}", some_integer);
} // 这里，some_integer 移出作用域。没有特殊之处

fn owner_resurn() {
    let s1 = gives_ownership(); // gives_ownership 将返回值移给 s1
    let s2 = String::from("hello"); // s2 进入作用域
    let s3 = takes_and_gives_back(s2); // s2 被移动到 takes_and_gives_back 中
                                       // 它也将返回值移给 s3
} // 这里 s3 移出作用域并被丢弃，s2 也移出作用域，但已经被移走，所以什么也不会发生，s1 移出作用域并被丢弃

fn gives_ownership() -> String {
    // gives_ownership 将返回值移动给调用它的函数
    let some_string = String::from("yours"); // some_string 进入作用域
    some_string // 返回 some_string 并移出给调用的函数
}

fn takes_and_gives_back(a_string: String) -> String {
    // a_string 进入作用域
    a_string // 返回 a_string 并移出给调用的函数
}

fn ownership_tuple() {
    let s1 = String::from("hello");
    let (s2, len) = calculate_length(s1);
    println!("The length of '{}' is {}", s2, len);
}

fn calculate_length(s: String) -> (String, usize) {
    let length = s.len(); // len() 返回字符串的长度
    (s, length)
}
