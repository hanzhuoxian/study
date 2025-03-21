use color::Colorize;

mod color;

fn main() {
    let hi = "Hello".red().on_yellow();
    println!("{}", hi);
    let hi = "Hello".on_yellow();
    println!("{}", hi);
    let hi = "Hello".red();
    println!("{}", hi);
    let hi = "Hello world".red().on_yellow().red();
    println!("{}", hi);
}
