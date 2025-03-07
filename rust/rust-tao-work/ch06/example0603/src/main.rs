fn modify(v: &mut [u32]) {
    v.reverse();
}

fn main() {
    let mut v = vec![1,2,3];
    modify(&mut v);
    println!("{:?}", v);
}
