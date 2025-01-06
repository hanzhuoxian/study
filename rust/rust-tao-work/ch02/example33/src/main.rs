fn move_coords(x: (i32, i32)) -> (i32, i32) {
    (x.0 + 1, x.1 + 1)
}
fn main() {
    let tuple: (&'static str, i32, char) = ("hello", 5, 'c'); // 定义元组
    assert_eq!(tuple.0, "hello"); // 使用下标访问元组
    assert_eq!(tuple.1, 5);
    assert_eq!(tuple.2, 'c');

    let coords = (0, 1);
    let result = move_coords(coords); // 使用元组返回多个值
    assert_eq!(result, (1, 2));

    let (x, y) = move_coords(coords); // 结构元组
    assert_eq!(x, 1);
    assert_eq!(y, 2);
}
