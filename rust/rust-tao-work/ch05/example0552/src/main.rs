use std::{
    cell::RefCell,
    rc::{Rc, Weak},
};

struct Node {
    data: i32,
    next: Option<Rc<RefCell<Node>>>,
    head: Option<Weak<RefCell<Node>>>,
}

impl Drop for Node {
    fn drop(&mut self) {
        println!("Dropping! {:?}", self.data);
    }
}
fn main() {
    let first = Rc::new(RefCell::new(Node {
        head: None,
        next: None,
        data: 1,
    }));

    let second = Rc::new(RefCell::new(Node {
        head: None,
        next: None,
        data: 2,
    }));

    let third = Rc::new(RefCell::new(Node {
        head: None,
        next: None,
        data: 3,
    }));

    first.borrow_mut().next = Some(second.clone());
    second.borrow_mut().head = Some(Rc::downgrade(&first));
    second.borrow_mut().next = Some(third.clone());
    third.borrow_mut().head = Some(Rc::downgrade(&first));
}
