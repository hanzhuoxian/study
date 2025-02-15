fn main() {
    let a = Box::new("hello");
    let b = Box::new("rust".to_string());

    let c = *a;
    let d = *b;

    println!("{:?}", a);
    // println!("{:?}", b);


}
