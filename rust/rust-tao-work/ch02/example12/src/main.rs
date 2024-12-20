
// #![feature(const_fn)]
const fn init_len() -> usize {
    return 5;
}
fn main() {
    let arr = [1; init_len()];
    println!("{} , {}", arr[2] , arr.len())
}
