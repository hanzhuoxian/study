use std::rc::Rc;

fn main() {
    let x = Rc::new(45);
    let y1 = x.clone();// 增加强引用计数 不会深复制，只是增加引用计数
    let y2 = x.clone();// 增加强引用计数
    println!("{:?}", Rc::strong_count(&x));
    let w = Rc::downgrade(&x);// 增加弱引用计数
    println!("{:?}", Rc::weak_count(&x));
    let y3 = &*x;// 不增加
    println!("{:?}", 100 - *x);
    println!("{:?}", Rc::strong_count(&x));
    println!("{:?}", Rc::weak_count(&x));
}
