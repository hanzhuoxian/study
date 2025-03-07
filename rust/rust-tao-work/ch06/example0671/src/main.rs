use core::{
    assert_eq,
    iter::{IntoIterator, Iterator},
};

fn main() {
    let a = [1, 2, 3];
    let mut iter = a.into_iter().map(|x| 2 * x);
    assert_eq!(iter.next(), Some(2));
    assert_eq!(iter.next(), Some(4));
    assert_eq!(iter.next(), Some(6));
    assert_eq!(iter.next(), None);
}
