struct MyStruct;

impl Copy for MyStruct {}

impl Clone for MyStruct {
    fn clone(&self) -> Self {
        *self
    }
}

#[derive(Copy, Clone)]
struct MyStruct2;

fn main() {
    println!("Hello, world!");
}
