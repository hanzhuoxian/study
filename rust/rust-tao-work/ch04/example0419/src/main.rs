use std::{cell::RefCell, rc::Rc};

type NodePtr<T> = Option<Rc<RefCell<Node<T>>>>;

struct Node<T> {
    data: T,
    next: NodePtr<T>,
}
fn main() {
    let first = Rc::new(RefCell::new(Node {
        data: 1,
        next: None,
    }));
    let second = Rc::new(RefCell::new(Node {
        data: 2,
        next: Some(first.clone()),
    }));
    first.borrow_mut().next = Some(second.clone());
    second.borrow_mut().next = Some(first.clone());
}
