use std::thread;

fn main() {
    let mut x = vec![1, 2, 3];
    thread::spawn(|| {
        x.push(4);
    });
    x.push(5);
    println!("{:?}", x);
}
