trait AnyTrait {
    fn any();
}
impl<T> AnyTrait for T {
    fn any() {
        println!("any");
    }
}

impl<T> AnyTrait for i32 {
    fn any() {
        println!("any copy");
    }
}
fn main() {}
