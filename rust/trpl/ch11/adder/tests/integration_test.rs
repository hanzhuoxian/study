use adder::add_two;

#[test]
fn it_works() {
    assert_eq!(2 + 2, 4);
}

#[test]
fn it_adds_two() {
    assert_eq!(4, add_two(2))
}
