fn main() {
    let outer_val = 1;
    let outer_sp = "hello".to_string();
    {
        let inner_val = 2;
        outer_val;
        outer_sp;
    }
    println!("{:?}", outer_val);
    // println!("{:?}", outer_sp); // borrow of moved value: `outer_sp` value borrowed here after move
    // println!("{:?}", inner_val); // cannot find value `inner_val` in this scope
}
