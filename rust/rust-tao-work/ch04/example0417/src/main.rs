type NodePtr<T> = Option<Box<Node<T>>>;

struct Node<T> {
    data: T,
    next: NodePtr<T>,
}
fn main() {
    let mut first = Box::new(Node{data: 1, next: None});
    let mut second = Box::new(Node{data: 2, next: None});
    first.next = Some(second);
    second.next = Some(first); 
}
