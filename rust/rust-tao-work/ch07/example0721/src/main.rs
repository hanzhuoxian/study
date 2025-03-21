#[derive(Debug)]
struct PrintDrop(&'static str);

impl Drop for PrintDrop {
    fn drop(&mut self) {
        println!("Dropping {}", self.0)
    }
}
fn main() {
    let x = PrintDrop("x");
    let y = PrintDrop("y");
    println!("{:?} {:?}", x, y);
    let tup1 = (PrintDrop("a"), PrintDrop("b"), PrintDrop("c"));
    let tup2 = (PrintDrop("x"), PrintDrop("y"), PrintDrop("z"));
    println!("{:?} {:?}", tup1, tup2);
}
