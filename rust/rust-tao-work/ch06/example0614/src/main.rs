fn hello() {
    println!("hello function pointer");
}
fn main() {
    let fn_prt: fn() = hello;
    println!("{:p}", fn_prt);
    let other_fn = fn_prt;
    println!("{:p}", other_fn);

    hello();
    other_fn();
    fn_prt();
    (fn_prt)();

}
