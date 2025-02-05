
fn main() {
    let a = "hello".to_string();
    let b = "world".to_string();
    let c = a + &b;
    println!("{:?}", c)
}
