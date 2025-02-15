use std::{rc::Rc, sync::Arc};

fn main() {
    let r = Rc::new("Rust".to_string());
    let a = Arc::new(vec![1,2,3,4,5]);
    // let x = *r;
    // let i = *a;
}
