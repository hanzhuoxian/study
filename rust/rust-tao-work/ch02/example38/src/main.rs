struct Empty;

fn main() {
    let x = Empty;
    println!("{:p}", &x);

    let y = x;
    println!("{:p}", &y);

    let z = Empty;
    println!("{:p}", &z);

    assert_eq!((..), std::ops::RangeFull)
}
