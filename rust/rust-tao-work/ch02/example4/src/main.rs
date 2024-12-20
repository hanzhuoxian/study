fn main() {
    let a = 1;
    // a = 2; //cannot assign twice to immutable variable

    let mut b = 2;
    println!("{b}");
    b = 3;

    println!("{a} , {b}")
}
