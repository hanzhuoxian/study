enum Color{
    Red = 0xff0000,
    Green = 0x00ff00,
    Blue = 0x0000ff,
}
fn main() {
    println!("roses are #{:06x}", Color::Red as i32);
    println!("violets are #{:06x}", Color::Blue as i32);
    println!("tree are #{:06x}", Color::Green as i32);
}
