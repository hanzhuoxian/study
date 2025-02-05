#[derive(Debug)]
struct Person {
    name: String,
}

impl Person {
    fn new<T>(name: T) -> Person
    where
        T: Into<String>,
    {
        Person { name: name.into() }
    }
}

fn main() {
    let person = Person::new("Alex");
    println!("{:?}", person);
    let person = Person::new("Alex".to_string());
    println!("{:?}", person.name);
}
