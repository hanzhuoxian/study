use std::error::Error;
use std::fs::File;
use std::io::Read;

type ParseResult<T> = Result<T, Box<dyn Error>>;

fn main() {
    let file_name = std::env::args().nth(1);
    match run(file_name) {
        Ok(sum) => println!("{}", sum),
        Err(e) => println!("{}", e),
    }
}

fn run(file_name: Option<String>) -> ParseResult<i32> {
    if file_name.is_none() {
        return Err("No file name provided".into());
    }
    let mut file = File::open(file_name.unwrap())?;
    let mut contents = String::new();
    file.read_to_string(&mut contents)?;
    let sum: i32 = contents
        .lines()
        .map(|line| line.parse::<i32>().unwrap())
        .sum();
    Ok(sum)
}
