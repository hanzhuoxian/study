fn main() {
    let s = String::from("hello, world");
    let s = s.chars().enumerate().map(|(i, c)|{
        if i % 2 == 0 {
            c.to_uppercase().to_string()
        } else {
            c.to_lowercase().to_string()
        }
    }).collect::<String>();

    println!("{:?}", s)
}
