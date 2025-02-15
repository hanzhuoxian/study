use std::borrow::Cow;

fn abs_all(input: &mut Cow<[i32]>) {
    for i in 0..input.len() {
        let v = input[i];
        if v < 0 {
            input.to_mut()[i] = -v
        }
    }
}
fn abs_sum(ns: &[i32]) -> i32 {
    let mut lst = Cow::from(ns);
    abs_all(&mut lst);
    lst.iter().fold(0, |acc, &n| acc + n)
}
fn main() {
    // 未发生克隆
    let s1 = [1,2,3];
    let mut i1 = Cow::from(&s1[..]);
    abs_all(&mut i1);
    println!("IN: {:?}", s1);
    println!("OUT: {:?}", i1);

    // 发生克隆
    let s2 = [1,2,3,-45,5];
    let mut i2 = Cow::from(&s2[..]);
    abs_all(&mut i2);
    println!("IN: {:?}", s2);
    println!("OUT: {:?}", i2);

    let mut v1 = Cow::from(vec![1,2,-3,4]);
    abs_all(&mut v1);
    println!("IN/OUT {:?}", v1);

    // 未发生克隆
    let s3 = [1,3,5,6];
    let sum1 = abs_sum(&s3[..]);
    println!("{:?}", s3);
    println!("{:?}", sum1);

    // 发生克隆
    let s4 = [1,-3, 5, -6];
    let sum2 = abs_sum(&s4);
    println!("{:?}", s4);
    println!("{:?}", sum2);
    


}
