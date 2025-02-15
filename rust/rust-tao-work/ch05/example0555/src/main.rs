use std::cell::RefCell;

fn main() {
    let x = RefCell::new(vec![1,2,3,4]);
    let mut mut_v = x.borrow_mut();
    mut_v.push(5);
    let mut mut_v2 = x.borrow_mut(); // already borrowed: BorrowMutError
    mut_v2.push(6);
}
