fn main() {
    let a = [1,2,3];
    let mut iter = a.iter();
    println!("{:?}", iter.size_hint());
    iter.next();
    println!("{:?}", iter.size_hint());
    iter.next();
    println!("{:?}", iter.size_hint());
    iter.next();
    println!("{:?}", iter.size_hint());
    iter.next();
    println!("{:?}", iter.size_hint());
}
