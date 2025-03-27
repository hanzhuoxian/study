use std::collections::{hash_map::Entry, HashMap};

fn main() {
    let mut book_reviews = HashMap::with_capacity(10);

    book_reviews.insert("Rust Book", "Good");
    book_reviews.insert("Rust Book2", "Bad");
    book_reviews.insert("Rust Book3", "Nice");
    for key in book_reviews.keys() {
        println!("{:?}", key);
    }
    for value in book_reviews.values() {
        println!("{:?}", value);
    }

    assert!(book_reviews.contains_key("Rust Book"));
    
    book_reviews.remove("Rust Book");

    let to_find = ["Rust Book3", "Rust Book2"];

    for book in &to_find {
        match book_reviews.get(book) {
            Some(review) => println!("{}: {}", book, review),
            None => println!("{} is unreviewed.", book),
        }
    }

    for (book, review) in &book_reviews {
        println!("{}: {}", book, review);
    }
    
}
