trait Page {
    fn set_page(&self, p: i32) {
        println!("Page default 1");
        println!("Page set: {:?}", p);
    }
}

trait PerPage {
    fn set_per_page(&self, num: i32) {
        println!("Per page default 10");
        println!("Per Page set: {:?}", num);
    }
}

struct MyPaginate {
    page: i32
}

impl Page for MyPaginate {}
impl PerPage for MyPaginate {}

fn main() {
    let my_paginate = MyPaginate { page: 1 };
    my_paginate.set_page(2);
    my_paginate.set_per_page(100);
    println!("{:?}", my_paginate.page);
}
