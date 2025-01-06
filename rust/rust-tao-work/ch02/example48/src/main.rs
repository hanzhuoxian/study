use std::collections::{BTreeSet, HashSet};

fn main() {
    let mut h_books = HashSet::new();
    h_books.insert("A Song of Ice and Fire");
    h_books.insert("The Emerald City");
    h_books.insert("The Odyssey");

    if !h_books.contains("The Emerald City") {
        println!(
            "We have {} book, but The Emerald City ain't one",
            h_books.len()
        )
    }
    println!("{:?}", h_books);

    let mut b_books = BTreeSet::new();
    b_books.insert("A Song of Ice and Fire");
    b_books.insert("The Emerald City");
    b_books.insert("The Odyssey");
    println!("{:?}", b_books);
}
