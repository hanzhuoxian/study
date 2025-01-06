fn main() {
    let arr: [i32; 3] = [1, 2, 3];
    let mut mut_arr = [1, 2, 3];
    assert_eq!(1, arr[0]);
    mut_arr[0] = 3;
    assert_eq!(3, mut_arr[0]);
    let init_arr = [0; 10]; // [0; 10] 创建值为 0 ， 长度 为 10 的数组。
    assert_eq!(0, init_arr[5]);
    assert_eq!(10, init_arr.len());
}
