fn main() {
    let s = String::from("hello 逗 world");
    let v = s
        .split(|c| (c as u32) >= (0x4E00 as u32) && (c as u32) <= (0x9FA5 as u32))
        .collect::<Vec<&str>>();
    println!("{:?}", v);

    let s = String::from("hello 逗 world");
    let v = s.split("逗").collect::<Vec<&str>>();
    println!("{:?}", v);

    let s = String::from("Mary had a little lambda");
    let v = s.splitn(3, " ").collect::<Vec<&str>>();
    println!("{:?}", v);

    let s = String::from("A.B.");
    let v = s.split('.').collect::<Vec<&str>>();
    println!("{:?}", v);

}
