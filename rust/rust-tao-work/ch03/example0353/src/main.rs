use std::{thread::{self, sleep}, time::Duration};

fn main() {
    let mut x = vec![1, 2, 3];

    println!("len {}", x.len());
    thread::spawn(move || {
        x.push(1);
        println!("after push len:{}", x.len());
    });
    // x.push(2);
}
