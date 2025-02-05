fn main() {
    let a: &'static str = "hello";
    let b = a as &str;
    let c: &'static str = b as &'static str;
    println!("{} {} {}", a, b, c);
}
