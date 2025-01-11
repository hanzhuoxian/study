fn foo() -> ! {
    loop {
        println!("jh")
    }
}
fn main() {
    let i = if false { foo() } else { 100 };
    assert_eq!(i, 100);
}
