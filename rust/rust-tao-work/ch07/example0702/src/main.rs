#[derive(Debug, Clone)]
struct Book {
    #[allow(dead_code)]
    name: String,
    isbn: i32,
    #[allow(dead_code)]
    version: i32,
}

fn main() {
    let book = Book {
        name: "Rust Programming".to_string(),
        isbn: 123456789,
        version: 1,
    };
    let book2 = Book { version: 2, ..book };
    println!("{:?}", book.isbn);
    println!("{:?}", book2);
}
