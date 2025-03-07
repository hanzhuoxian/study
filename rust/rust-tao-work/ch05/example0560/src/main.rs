fn xy<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() % 2 == 0 {
        x
    } else {
        y
    }
}
fn main() {
    let x = String::from("helloo");
    let z;
    let y = String::from("world");
    z = xy(&x, &y);
    println!("{:?}", z);
}
