
trait Add<RHS = Self> {
    // RHS = Self 为类型参数指定默认值，Self 是每个 trait 都带有的隐式类型参数，代表实现当前 trait 的具体类型
    type Output; // 关联类型
    fn add(self, rhs: RHS) -> Self::Output;
    fn sub(self, rhs: RHS) -> Self::Output;
}

impl Add<i32> for i32 {
    type Output = i32;
    fn add(self, rhs: i32) -> i32 {
        self + rhs
    }
    fn sub(self, rhs:i32) -> i32 {
        self - rhs
    }
}

impl Add<u32> for u32 {
    type Output = i32;
    fn add(self, i: u32) -> Self::Output {
        (self + i) as i32
    }
    fn sub(self, rhs: u32) -> Self::Output {
        (self - rhs) as i32
    }
}

fn main() {
    let (a, b, c, d) = (1i32, 2i32, 3u32, 4u32);
    let x: i32 = a.add(b);
    let y: i32 = c.add(d);
    assert_eq!(x, 3);
    assert_eq!(y, 7);
    assert_eq!(a.sub(b), -1);
    assert_eq!(d.sub(c), 1);
}
