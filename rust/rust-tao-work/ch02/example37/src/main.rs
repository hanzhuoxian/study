struct Integer(u32); // 相当于把类型 u32 包装成了一个新类型

type Int = i32; // 使用 type 为类型创建别名

fn main() {
    let int = Integer(10);
    assert_eq!(int.0, 10);

    let int: Int = 10;
    assert_eq!(int, 10);
}
