trait Foo<'a> {}
struct FooImpl<'a> {
    s: &'a [u32],
}

impl<'a> Foo<'a> for FooImpl<'a> {}

// 编译报错
// fn foo<'a>(s: &'a [u32]) -> Box<dyn Foo<'a>> {
//     Box::new(FooImpl{s: s})
// }

fn foo_a<'a>(s: &'a [u32]) -> Box<dyn Foo<'a> + 'a> {
    Box::new(FooImpl{s: s})
}

fn main() {
}
