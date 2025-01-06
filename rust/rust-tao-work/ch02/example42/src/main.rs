fn main() {
    let s = Some(42);
    let num = s.unwrap();
    println!("num is {}", num);
    
    match s {
        Option::Some(n) => println!("num is {}", n),
        Option::None => (),
    }
}
