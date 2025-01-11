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

trait Paginate: Page + PerPage {
    fn set_skip_page(&self, num: i32) {
        println!("Skip page: {:?}", num)
    }
}

struct MyPaginate {
    page: i32,
}

impl Page for MyPaginate {}
impl PerPage for MyPaginate {}
// 为所有拥有 Page 和 PerPage 行为的类型实现 Paginate。
impl<T: Page + PerPage> Paginate for T {}

fn main() {
    let my_paginate = MyPaginate { page: 1 };
    my_paginate.set_page(2);
    my_paginate.set_per_page(100);
    my_paginate.set_skip_page(5);
    println!("{:?}", my_paginate.page);
}
