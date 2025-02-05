use std::thread;
// 多线程间共享不可变引用
fn main() {
    let x = vec![1, 2, 3, 4];
    thread::spawn(move || {
        println!("{}", x.len())
    });
}
