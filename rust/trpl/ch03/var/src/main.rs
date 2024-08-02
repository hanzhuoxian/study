fn main() {
    // 不可变变量
    let x = 5;
    println!("The value of x is: {}", x);
    // x = 6; // 编译错误 cannot assign twice to immutable variable

    let mut x = 5; // 隐藏前面的 x 变量
    println!("The value of mut x is: {}", x);
    x = 6;
    println!("The value of x mut is: {}", x);

    const MAX_POINTS: u32 = 100_000;
    // MAX_POINTS = 100; // 编译错误 error[E0070]: invalid left-hand side of assignment
    println!("The value of MAX_POINTS is: {}", MAX_POINTS);
}
