pub mod outer_mod {
    // outer_mod 及子模块可见
    pub(self) fn outer_mod_fn() {}
    pub mod inner_mod {
        // 只能在指定的模块中访问
        pub(in crate::outer_mod) fn outer_mod_visible_fn() {}
        pub fn pub_visible_fn() {}
        // 对整个 crate 可见
        pub(crate) fn crate_mod_visible_fn() {}
        // 只对当前父模块可见
        pub(super) fn super_mod_visible_fn() {
            inner_mod_fn();
            super::outer_mod_fn();
        }
        pub(super) fn in_super_mod_visible_fn() {}
        // 只对本模块可见
        pub(self) fn inner_mod_fn() {}
    }

    pub fn foo() {
        inner_mod::crate_mod_visible_fn();
        inner_mod::outer_mod_visible_fn();
        inner_mod::super_mod_visible_fn();
        inner_mod::in_super_mod_visible_fn();
    }
}

fn bar() {
    outer_mod::inner_mod::crate_mod_visible_fn();
    // outer_mod::inner_mod::super_mod_visible_fn();
    // outer_mod::inner_mod::outer_mod_visible_fn();
    outer_mod::foo();
    outer_mod::inner_mod::pub_visible_fn();
}
fn main() {
    bar();
}
