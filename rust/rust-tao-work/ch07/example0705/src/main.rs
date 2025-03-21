use color::Colorize;

mod color;

fn main() {
    let red = "red".red();
    println!("{}", red);
    let blue = "blue".blue();
    println!("{}", blue);
    let yellow = "yellow".yellow();
    println!("{}", yellow);
    let yellow_on_blue = "yellow_on_blue".yellow().on_blue();
    println!("{}", yellow_on_blue);
    let red = "red".color("red");
    println!("{}", red);
    let yellow = "yellow".on_color("yellow");
    println!("{}", yellow);
    let hi = "on_red".on_red();
    println!("{}", hi);
    let hi = "on_red".on_yellow();
    println!("{}", hi);

}
