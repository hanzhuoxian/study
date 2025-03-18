#[derive(Debug, Clone, Copy)]
struct Book<'a> {
    name: &'a str,
    isbn: i32,
    version: i32,
}

fn main() {
    let book = Book {
        name: "Rust Programming",
        isbn: 123456789,
        version: 1,
    };
    let book2 = Book { version: 2, ..book };
    println!(
        "{:?}, {:?}, {:?} {:?}",
        book.name, book.isbn, book.version, book
    );
    println!("{:?}", book2);
}
