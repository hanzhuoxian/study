fn main() {
    let mut v = [-5, 4, 1, -3, 2];
    v.sort();
    println!("{:?}", v);
    assert_eq!(v, [-5, -3, 1, 2, 4]);
    v.sort_by(|a, b| a.cmp(b));
    assert_eq!(v, [-5, -3, 1, 2, 4]);
    v.sort_by(|a, b| b.cmp(a));
    println!("{:?}", v);
    assert_eq!(v, [4, 2, 1, -3, -5]);
}
