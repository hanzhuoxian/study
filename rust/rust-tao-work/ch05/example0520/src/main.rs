fn change_first(mut v: [i32; 3]) -> [i32; 3] {
    v[0] = 3;
    v
}
fn main() {
    let v = [1, 2, 3];
    let cv = change_first(v);
    assert_eq!([1,2,3], v);
    assert_eq!([3,2,3], cv);

}
