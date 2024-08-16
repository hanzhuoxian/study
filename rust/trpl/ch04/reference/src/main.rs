fn main() {
    let mut s1 = String::from("hello");

    let len = calculate_length(&s1);

    println!("The length of {s1} is {len}.");

    change(&mut s1);
    change(&mut s1);
    println!("change after s1 is : {s1}");

    let r1 = &s1;
    let r2 = &s1;
    println!("r1 {r1}, r2 {r2}");

    let mut s1 = String::from("rust");
    let w1 = &mut s1;
    // let w2 = &mut s1; // rustc: cannot borrow `s1` as mutable more than once at a time second mutable borrow occurs here
    println!("w1 {w1}");

    let mut s1 = String::from("rust mut");
    let w1 = &mut s1;
    println!("{w1}");
    let w2 = &mut s1;
    w2.push_str(" w2"); // no problem
    println!("{w2}")
}

// 编译错误
// fn dangling() -> &String {
//     let s = String::from("danglilng");
//     &s
// }

fn change(s: &mut String) {
    s.push_str(",world!")
}

// 以对象的引用作为参数，而不获取对象的所有权 s 指向 s1
fn calculate_length(s: &String) -> usize {
    return s.len();
}
