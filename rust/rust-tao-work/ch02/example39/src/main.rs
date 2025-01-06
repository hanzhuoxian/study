enum Number { // 类型
    Zero, // 值
    One,
    Two,
}

fn number_print(num: Number) {
    match num {
        Number::Zero => println!("0"),
        Number::One => println!("1"),
        Number::Two => println!("2"),
    }
}

fn main() {
    let zero = Number::Zero;
    number_print(zero);
    let one = Number::One;
    number_print(one);
    let two = Number::Two;
    number_print(two);
}
