use std::{fs::File, io::{Error, Read}};

fn main() -> Result<(), Error>{
    let mut f = File::open("./bat.txt")?;
    
    let mut buf: Vec<u8> = vec![];
    let size = f.read_to_end(&mut buf)?;
    println!("{:?}", size);

    Ok(())
}
