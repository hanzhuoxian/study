fn main() {
    println!("Hello, world!");

    another_function();
    another_function_x(5);
    print_labeled_measurement(5, 'h');

    let y = 6;
    // let x = (let y=6); let y=6 是一个语句不是一个表达式不能赋值给 x
    println!("{}", y);

    let z = {
        let x = 3;
        // x+1 结尾没有分号是一个表达式，如果加上分号就变成语句
        x + 1
    };

    println!("z = {}", z);

    let x = five();
    println!("The value x is: {}", x);

    let x = plus_one(5);
    println!("The value x is: {}", x);
}

fn another_function() {
    println!("Another function.")
}

fn another_function_x(x: i32) {
    println!("The value of x is {}.", x)
}

fn print_labeled_measurement(value: i32, unit_label: char) {
    println!("The measurement is: {}{}", value, unit_label)
}

// 具有返回值的函数
//
// 函数的返回值等于最后一个表达式的值，也可以 return 提前返回
fn five() -> i32 {
    5
}

fn plus_one(x: i32) -> i32 {
    x + 1
}
