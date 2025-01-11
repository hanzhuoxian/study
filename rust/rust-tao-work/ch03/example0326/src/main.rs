trait Add<RHS = Self> {
    // RHS = Self 为类型参数指定默认值，Self 是每个 trait 都带有的隐式类型参数，代表实现当前 trait 的具体类型
    type Output; // 关联类型
    fn add(self, rhs: RHS) -> Self::Output;
}

impl Add<&str> for String {
    type Output = String;
    fn add(mut self, rhs: &str) -> String {
        self.push_str(rhs);
        self
    }
}

fn main() {
    let hello = String::from("hello");
    assert_eq!(hello.add(",world!"), "hello,world!");
}
