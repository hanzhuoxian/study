fn main() {
    let mut v1 = vec![];
    v1.push(1);
    v1.push(2);
    v1.push(3);
    assert_eq!(v1, [1, 2, 3]);
    assert_eq!(v1[1], 2);

    let mut v2 = vec![0;10];
    assert_eq!(v2[9], 0);
    v2[9] = 1;
    assert_eq!(v2[9], 1);

    let mut v3 = Vec::new();
    v3.push(4);
    v3.push(5);
    v3.push(6);

    let v4 = vec!(3;3);
    assert_eq!(v4, [3,3,3]);

    // println!("{}", v3[4]);
}
