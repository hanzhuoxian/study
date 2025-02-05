fn test_copy<T: Copy>(i: T) {
    println!("hhh");
}
fn main() {
    let a = 1;
    test_copy(a);
    let a = "String".to_string();
    test_copy(a);
}
