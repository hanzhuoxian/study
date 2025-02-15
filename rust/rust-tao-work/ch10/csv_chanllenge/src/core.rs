pub mod read;
pub mod write;

use std::{fs::File, path::PathBuf};

use crate::err::Error;

pub fn open(path: PathBuf) -> Result<File, Error>{
    let file = File::open(path)?;
    Ok(file)
}
