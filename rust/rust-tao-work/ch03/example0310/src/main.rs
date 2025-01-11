enum Void {}
fn main() {
    let res: Result<u32, Void> = Ok(0);
    let Ok(num) = res;
    println!("{:?}", num)
}
