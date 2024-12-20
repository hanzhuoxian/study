fn main() {
    let place1 = "hello";
    let place2 = "hello".to_string();
    let other = place1; // place1 将内存地址转移给 other
    println!("{:?} , {:?}", other, place1);
    let other = place2;
    println!("{:?}", other); // place2 将内存地址转移给 other
                             // println!("{:?}", place2); // place2 value borrowed here after move
}
