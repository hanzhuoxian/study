use std::num::ParseIntError;

fn square(number_str: &str) -> Result<i32, ParseIntError> {
    number_str.parse::<i32>().map(|n| n.pow(2))
}

type ParseResult<T> = Result<T, ParseIntError>;
fn type_square(number_str: &str) -> ParseResult<i32> {
    number_str.parse::<i32>().map(|n| n.pow(2))
}

fn main() {
    match square("10") {
        Ok(n) => assert_eq!(n, 100),
        Err(err) => println!("{:?}", err),
    }
    match type_square("10") {
        Ok(n) => assert_eq!(n, 100),
        Err(err) => println!("{:?}", err),
    }
}
