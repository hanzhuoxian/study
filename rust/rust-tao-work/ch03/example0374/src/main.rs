use std::ops::Add;

#[derive(Debug, PartialEq, PartialOrd)]
struct Int(i32);

impl Add<i32> for Int {
    type Output = i32;

    fn add(self, rhs: i32) -> Self::Output {
        self.0 + rhs
    }
}

// impl Add<i32> for Option<Int> {
//     type Output = i32;

//     fn add(self, rhs: i32) -> Self::Output {
//         self.unwrap_or(Int(0)) + rhs
//     }
// }

impl Add<i32> for Box<Int> {
    type Output = i32;

    fn add(self, rhs: i32) -> Self::Output {
        *self + rhs
    }
}

fn main() {
    assert_eq!(Int(1) + 2, 3);
    assert_eq!(Box::<Int>::new(Int(1)) + 2, 3);
}
