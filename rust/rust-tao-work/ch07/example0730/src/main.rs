#[derive(Debug)]
struct PrintDrop(&'static str);

impl Drop for PrintDrop {
    fn drop(&mut self) {
        println!("Dropping {}", self.0)
    }
}
fn main() {
    let z = PrintDrop("z");
    let x = PrintDrop("x");
    let y = PrintDrop("y");
    let closure = move || {
        y;
        z;
        x;
    };

    let z = PrintDrop("z");
    let x = PrintDrop("x");
    let y = PrintDrop("y");
    let closure = move || {
        {
            let z_ref = &z;
        }
        x;
        y;
        z;
    };
}
