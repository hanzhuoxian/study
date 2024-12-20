fn main() {
    let x = &temp();
    println!("{x}")
    // temp() = *x; // E0070 cannot assign to this expression
}

//
pub fn temp() -> i32 {
    return 1;
}
