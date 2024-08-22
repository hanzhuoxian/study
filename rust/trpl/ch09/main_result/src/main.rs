use std::{error::Error, fs::File};

fn main() -> Result<(), Box<dyn Error>> {
    let file = File::open("hello.txt")?;
    file.metadata()?;
    Ok(())
}
