fn change_first(v: &mut [i32; 3]) {
    v[0] = 3;
}
fn main() {
    let mut a = [1, 2, 3];
    change_first(&mut a);
    assert_eq!([3, 2, 3], a);
}
