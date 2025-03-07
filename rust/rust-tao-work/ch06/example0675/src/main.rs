use core::assert_eq;

fn main() {
    let a = [1, 2, 3];
    for v in a.iter().rev() {
        println!("{:?}", v);
    }
    let mut iter = a.iter().rev();
    assert_eq!(iter.next(), Some(&3));
    assert_eq!(iter.next(), Some(&2));
    assert_eq!(iter.next(), Some(&1));
    assert_eq!(iter.next(), None);
}
