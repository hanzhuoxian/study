fn main() {
    let hello = "hello";
    let c = || println!("{:?}", hello);
    c();
    c();
    println!("{:?}", hello);
}
