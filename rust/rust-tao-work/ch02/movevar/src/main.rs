fn main() {
    let place1 = "hello";
    let place2 = "hello".to_string();
    let other  = place1;
    println!("other {:?}", other);
    let other = place2;
    println!("other {:?}", other);
}
