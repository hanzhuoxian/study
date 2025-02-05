// 对象不安全 trait
#[allow(dead_code)]
trait Foo {
    fn bad<T>(&self, x: T);
    fn new() -> Self;
}

// 对象安全 trait，将不安全的 trait 方法拆分出去
#[allow(dead_code)]
trait FooSafe {
    fn bad<T>(&self, x: T);
}

#[allow(dead_code)]
trait FooUnsafe {
    fn new() -> Self;
}

// 对象安全的 trait ，使用 where 子句增加 trait 限定，
trait Foo1 {
    fn bad<T>(&self, x: T);
    fn new() -> Self
    where
        Self: Sized;
}

fn main() {
    println!("Hello, world!");
}
