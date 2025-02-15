use std::io;

#[derive(Debug)]
pub enum Error {
    IO(io::Error),
    Program(&'static str),
}

impl From<io::Error> for Error {
    fn from(e: io::Error) -> Error {
        Error::IO(e)
    }
}

impl From<&'static str> for Error {
    fn from(e: &'static str) -> Error{
        Error::Program(e)
    }
}
