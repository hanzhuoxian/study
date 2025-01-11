trait Add<RHS, Output> {
    // RHS 代表加法操作符右侧的类型 Output 代表返回类型
    fn add(self, rhs: RHS) -> Output; // self 代表 实现该 trait 的类型
    fn sub(self, rhs: RHS) -> Output;
}

impl Add<i32, i32> for i32 {
    fn add(self, rhs: i32) -> i32 {
        self + rhs
    }
    fn sub(self, rhs: i32) -> i32 {
        self - rhs
    }
}

impl Add<u32, i32> for u32 {
    fn add(self, rhs: u32) -> i32 {
        (self + rhs) as i32
    }
    fn sub(self, rhs: u32) -> i32 {
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
