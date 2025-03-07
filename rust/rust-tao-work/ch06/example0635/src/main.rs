fn main() {
    let s = "hello";
    let c = move || s;
    c();
    c();
    println!("{:?}", s);
}
