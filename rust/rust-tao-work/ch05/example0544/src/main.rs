fn main() {
    let x = Box::new("hello");
    let y = x;
    println!("{y:?}");
    // println!("{x:?}");
}
