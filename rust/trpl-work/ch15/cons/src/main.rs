enum List {
   Cons(i32, Box<List>),
   Nil
}
use crate::List::{Cons, Nil};

fn main() {
    let list = new(5);
    print_list(list);
}

fn new(c:i32) -> List {
    if c == 1 {
        return  Cons(1, Box::new(Nil));
    }

    Cons(c, Box::new(new(c-1)))
}

fn print_list(list : List) {
    match list {
        Nil => {}
        Cons(i, next) => {
            println!("{i}");
            print_list(*next);
        } 
    }
}
