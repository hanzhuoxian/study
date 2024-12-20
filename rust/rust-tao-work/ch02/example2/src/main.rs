//  extern crate std;  // 声明语句 Rust 自动引入
// use std::prelude::v1::*; // 声明语句 Rust 自动引入

fn main() {
    pub fn answer() -> () {
        let a = 40;
        let b = 2;
        assert_eq!(sum(a, b), 43)
    }

    pub fn sum(a: i32, b: i32) -> i32 {
        a + b
    }
    answer();

    // fn temp() -> i32 {
    //     1
    // }

    // let x = temp();
    // temp() = &x
}
