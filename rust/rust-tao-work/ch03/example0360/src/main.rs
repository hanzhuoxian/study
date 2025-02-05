fn foo(f :&[i32]) {
    println!("{:?}", f)
}
fn main() {
    let v = vec![1,2,3];
    foo(&v);
}
