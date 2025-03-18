use color::Colorize;

mod color;

fn main() {
    let hi = "Hello".red().on_yellow();
    println!("{}", hi);
}
