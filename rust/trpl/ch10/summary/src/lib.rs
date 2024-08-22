use std::fmt::Display;

// 定义一个 trait，类似于其他语言中的接口, 学完之后感觉更像父类
pub trait Summary {
    fn summarize(&self) -> String;
    fn read_more(&self) -> String {
        String::from("read more...")
    }
}

pub struct NewArticle {
    pub headline: String,
    pub location: String,
    pub author: String,
    pub content: String,
}

impl Summary for NewArticle {
    fn summarize(&self) -> String {
        format!("{}, by {} ({})", self.headline, self.author, self.location)
    }
}

pub struct Tweet {
    pub username: String,
    pub content: String,
    pub reply: bool,
    pub retweet: bool,
}

impl Summary for Tweet {
    fn summarize(&self) -> String {
        format!("{}: {}", self.username, self.content)
    }
}

// trait 作为参数
pub fn notify(item: &impl Summary) {
    println!("Breaking new! {}", item.summarize());
}

// trait bound 语法
pub fn notify_t<T: Summary>(item: &T) {
    println!("Breaking new! {}", item.summarize());
}

// item1 与 item2 具体类型可以不同
pub fn notify_two(item1: &impl Summary, item2: &impl Summary) {
    println!("Breaking new! {} {}", item1.summarize(), item2.summarize());
}
// item1 与 item2 具体类型必须相同
pub fn Notify_two_t<T: Summary>(item1: &T, item2: &T) {
    println!("Breaking new! {} {}", item1.summarize(), item2.summarize());
}

// 使用 + 标识同时实现
pub fn notify_plus<T: Summary + Display>(item: &T) {
    println!("{item}");
}

// 使用 where 语法
pub fn notify_plus_v1<T>(item: &T)
where
    T: Summary + Display,
{
    println!("{item}");
}

// 返回 trait

pub fn returns_summarize() -> impl Summary {
    Tweet {
        username: String::from("horse_ebooks"),
        content: String::from("of course, as you probably already know, people"),
        reply: false,
        retweet: false,
    }
}

pub fn returns_summarize_switch(switch: bool) -> impl Summary {
    // if switch {
    Tweet {
        username: String::from("horse_ebooks"),
        content: String::from("of course, as you probably already know, people"),
        reply: false,
        retweet: false,
    }
    // } else { // 编译报错
    //     NewArticle {
    //         headline:String::from("rust"),
    //         content: String::from("rust is master"),
    //         author:String::from("zhuo xian"),
    //         location:String::from("zh-cn"),
    //     }
    // }
}

pub struct Pair<T> {
    x: T,
    y: T,
}

impl<T> Pair<T> {
    pub fn new(x: T, y: T) -> Self {
        Self { x, y }
    }
}

// 有条件的实现方法
impl<T: Display + PartialOrd> Pair<T> {
    pub fn cmp_display(&self) {
        if self.x > self.y {
            println!("The largest member is x = {}", self.x);
        } else {
            println!("The largest member is y = {}", self.y);
        }
    }
}

// 标准库的默认实现
// impl<T: fmt::Display + ?Sized> ToString for T {
//     // A common guideline is to not inline generic functions. However,
//     // removing `#[inline]` from this method causes non-negligible regressions.
//     // See <https://github.com/rust-lang/rust/pull/74852>, the last attempt
//     // to try to remove it.
//     #[inline]
//     default fn to_string(&self) -> String {
//         let mut buf = String::new();
//         let mut formatter = core::fmt::Formatter::new(&mut buf);
//         // Bypass format_args!() to avoid write_str with zero-length strs
//         fmt::Display::fmt(self, &mut formatter)
//             .expect("a Display implementation returned an error unexpectedly");
//         buf
//     }
// }
