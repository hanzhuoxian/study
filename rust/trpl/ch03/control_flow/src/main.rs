fn main() {
    // if 示例
    if_fn();
    // loop 示例
    loop_fn();
    // while 示例
    while_fn();
    // for 示例
    for_fn();
}

fn if_fn() {
    println!("==========================if start=========================");
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
    println!("==========================if end=========================");
}

fn while_fn() {
    println!("==========================while start=========================");
    let mut i = 5;
    while i != 0 {
        println!("i : {i}");
        i -= 1;
    }
    println!("==========================while end=========================");
}

fn for_fn() {
    println!("==========================for start=========================");

    let a = [1, 2, 3, 4, 5];
    for element in a {
        println!("{element}");
    }

    for element in (1..5).rev() {
        println!("{element}");
    }
    println!("==========================for end=========================");
}
fn loop_fn() {
    println!("==========================loop start=========================");
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
    println!("The result is {result}");

    let mut count = 0;
    'counting_up: loop {
        println!("count = {count}");
        let mut remaining = 10;

        loop {
            println!("remaining = {remaining}");
            if remaining == 9 {
                break;
            }

            if count == 2 {
                break 'counting_up;
            }

            remaining -= 1;
        }
        count += 1;
    }
    println!("End count = {count}");
    println!("==========================loop end=========================");
}
