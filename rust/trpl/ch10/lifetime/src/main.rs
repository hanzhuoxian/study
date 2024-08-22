use std::fmt::Display;

fn main() {
    // 声明周期避免悬垂引用
    let r; // ---------+-- 'a
    {
        let x = 5; // --+-- 'b
        r = &x; // --+-- end 'b
    }
    // println!("r: {}", r);

    let string1 = String::from("abcd");
    let result;
    {
        let str2 = "xyz";
        result = longest(string1.as_str(), str2);

        println!("The longest string is {result}");

        let string2: String = String::from("abcd");
        // result = longest(string1.as_str(), string2.as_str());

        println!("The longest string is {result}");
    }

    println!("The longest string is {result}");

    let novel = String::from("Call me Ishmael. Some years ago...");
    let first_sentence = novel.split('.').next().unwrap();
    let i = ImportantExcerpt {
        part: first_sentence,
    };

    let s: &'static str = "I have a static lifetime.";
} // ---------+ end 'a

// 函数中的泛型生命周期

fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() > y.len() {
        x
    } else {
        y
    }
}

// 生命周期语法是用于将函数的多个参数与其返回值的生命周期进行关联的。一旦它们形 成了某种关联，Rust 就有了足够的信息来允许内存安全的操作并阻止会产生悬垂指针亦或是 违反内存安全的行为。

// 这个注解意味着 ImportantExcerpt 的实例不能比其 part 字段中的引用存在 的更久。
struct ImportantExcerpt<'a> {
    part: &'a str,
}

// 第一条规则是编译器为每一个引用参数都分配一个生命周期参数。换句话说就是，函数有一个 引用参数的就有一个生命周期参数:fn foo<'a>(x: &'a i32) ，有两个引用参数的函数就有两 个不同的生命周期参数，fn foo<'a, 'b>(x: &'a i32, y: &'b i32) ，依此类推。
// 第二条规则是如果只有一个输入生命周期参数，那么它被赋予所有输出生命周期参数:fn foo<'a>(x: &'a i32) -> &'a i32 。
// 第三条规则是如果方法有多个输入生命周期参数并且其中一个参数是&self 或&mut self ， 说明是个对象的方法 (method)(译者注:这里涉及 rust 的面向对象参见 17 章)，那么所有输出 生命周期参数被赋予 self 的生命周期。第三条规则使得方法更容易读写，因为只需更少的符号。

impl<'a> ImportantExcerpt<'a> {
    fn level(&self) -> i32 {
        3
    }
}

impl<'a> ImportantExcerpt<'a> {
    fn announce_and_return_part(&self, announcement: &str) -> &str {
        println!("Attention please: {}", announcement);
        self.part
    }
}

fn longest_with_an_announcement<'a, T: Display>(x: &'a str, y: &'a str, ann: T) -> &'a str {
    println!("Announcement! {}", ann);
    if x.len() > y.len() {
        x
    } else {
        y
    }
}
