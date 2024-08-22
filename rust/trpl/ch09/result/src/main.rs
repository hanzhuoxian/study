use std::fs::{self, File};
use std::io::Read;

fn main() {
    match read_user_name_from_file_v4() {
        Ok(user_name) => println!("Hello, {}!", user_name),
        Err(error) => println!("Error reading user name: {:?}", error),
    }

    let file_name = String::from("hello.txt");

    let greet_file_result = File::open(&file_name);
    let greet_file = match greet_file_result {
        Ok(file) => file,
        Err(error) => match error.kind() {
            std::io::ErrorKind::NotFound => match File::create(&file_name) {
                Ok(fc) => fc,
                Err(e) => panic!("Problem creating the file: {:?}", e),
            },
            other_error => panic!("Problem opening the file: {:?}", other_error),
        },
    };

    // 使用 unwrap 或 expect 简化代码
    // 返回 ok 中的值，或者 panic
    let greet_file = File::open(&file_name).unwrap();
    let greet_file = File::open(&file_name).expect("Failed to open hello.txt");
}

fn read_user_name_from_file() -> Result<String, std::io::Error> {
    let file_name = String::from("hello.txt");
    let file_result = File::open(&file_name);
    let mut file = match file_result {
        Ok(file) => file,
        Err(error) => return Err(error),
    };

    let mut user_name = String::new();
    match file.read_to_string(&mut user_name) {
        Ok(_) => Ok(user_name),
        Err(error) => Err(error),
    }
}

// 传播错误简写 ? 运算符
fn read_user_name_from_file_v2() -> Result<String, std::io::Error> {
    let mut file = File::open(String::from("hello.txt"))?;
    let mut user_name = String::new();
    file.read_to_string(&mut user_name)?;
    Ok(user_name)
}

// ? 运算符 链式操作
fn read_user_name_from_file_v3() -> Result<String, std::io::Error> {
    let mut user_name = String::new();
    File::open(String::from("hello.txt"))?.read_to_string(&mut user_name)?;
    Ok(user_name)
}

fn read_user_name_from_file_v4() -> Result<String, std::io::Error> {
    fs::read_to_string(String::from("hello.txt"))
}

fn last_char_of_first_line(text: &str) -> Option<char> {
    text.lines().next()?.chars().last()
}
