use summary::{notify, returns_summarize, NewArticle, Pair, Summary, Tweet};

fn main() {
    let tweet = Tweet {
        username: String::from("horse_ebooks"),
        content: String::from("of course, as you probably already know, people"),
        reply: false,
        retweet: false,
    };
    println!("1 new tweet: {}", tweet.summarize());

    let article = NewArticle {
        headline: String::from("rust"),
        content: String::from("rust is master"),
        author: String::from("zhuo xian"),
        location: String::from("zh-cn"),
    };
    println!(
        "1 new article: {} {}",
        article.summarize(),
        article.read_more()
    );

    notify(&article);
    notify(&article);

    let s = returns_summarize();
    println!("returns_summarize: {}", s.summarize());

    let pair = Pair::new(1, 2);
    pair.cmp_display();
}
