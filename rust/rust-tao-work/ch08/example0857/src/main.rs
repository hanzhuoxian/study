use std::{collections::HashMap, hash::Hash};

fn main() {
    let mut book_reviews = HashMap::new();
    book_reviews.insert("Rust Book", "Good");

    let mut book_reviews1 = HashMap::new();
    book_reviews1.insert("Golang Book", "Good");

    book_reviews.extend(book_reviews1);
    println!("{:?}", book_reviews);

    let mut book_review2 = HashMap::new();
    book_review2.insert("PHP book", "Very Good");
}
