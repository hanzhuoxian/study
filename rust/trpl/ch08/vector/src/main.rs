fn main() {
    let mut v: Vec<i32> = Vec::new(); // vector 创建
    v.push(1); // vector 增加值
    v.push(2);
    v.push(3);

    let third: &i32 = &v[2];

    // v.push(4); //cannot borrow `v` as mutable because it is also borrowed as immutable mutable borrow occurs here

    println!("The third element is : {}", third);

    let third: Option<&i32> = v.get(2);
    match third {
        Some(third) => println!("The third element is : {}", third),
        None => println!("There is no third element"),
    }

    let v = vec![1, 2, 3]; // 使用宏创建初始值的 vector

    // 遍历 vector 中的元素
    for i in &v {
        println!("{i}")
    }

    let mut v = vec![100, 32, 57];
    for i in &mut v {
        *i += 50;
    }
    for i in &v {
        println!("{i}")
    }

    enum SpreadsheetCell {
        Int(i32),
        Float(f64),
        Text(String),
    }

    let row = vec![
        SpreadsheetCell::Int(3),
        SpreadsheetCell::Float(10.12),
        SpreadsheetCell::Text(String::from("blue")),
    ];

    {
        let v = vec![1, 2, 3];

        println!("{}", &v[0]);
    } // <- v goes out of scope and is freed here
}
