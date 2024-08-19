mod front_and_house {
    pub mod hosting {
        pub fn add_to_wait_list() {} // 使用 pub 将函数公有
        pub fn seat_at_table() {}
    }

    mod serving {
        fn take_order() {}
        fn server_order() {}
        fn take_payment() {}
    }
}

fn eat_at_restaurant() {
    // 绝对路径
    crate::front_and_house::hosting::add_to_wait_list();
    // 相对路径
    front_and_house::hosting::add_to_wait_list();
}
