fn main() {
    let number = 6;

    if number < 5 {
        println!("condition was true");
    } else {
        println!("condition was false");
    }

    if number != 0 {
        println!("number was something other than zero");
    }

    let number = 6;
    if number % 4 == 0 {
        println!("number is divisible by 4");
    } else if number % 3 == 0 {
        println!("number is divisible by 3");
    } else if number % 2 == 0 {
        println!("number is divisible by 2");
    } else {
        println!("number is not divisible by 4,3, or 2");
    }

    let condition = true;
    // if 语句表达式可以用于赋值
    let number = if condition { 5 } else { 6 };
    println!("The value of number is : {number}");
    // error[E0308]: `if` and `else` have incompatible types
    // let number = if condition {5} else {"six"};

    loop {
        println!("again!");
        break;
    }

    let mut counter = 6;
    let result = loop {
        counter += 1;
        if counter == 10 {
            break counter * 2;
        }
    };
    println!("The result is {result}")
}
