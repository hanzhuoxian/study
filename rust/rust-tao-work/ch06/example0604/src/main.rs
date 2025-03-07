fn f() {
    println!("1")
}
fn main() {
    f();
    {
        f();
        fn f() {
            println!("3")
        }
    }

    f();
    fn f() {
        println!("2")
    }
}
