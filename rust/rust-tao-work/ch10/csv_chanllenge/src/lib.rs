//! This is a document for the csv_challenge crate.
//! 
//! Usage:
//! ```ignore
//! use csv_challenge::{
//! Opt,
//! load_csv,write_csv,
//! replace_column,
//! };
//! ```
//! 

mod opt;
mod err;
mod core;

pub use opt::Opt;
pub use self::core::{
    read::{load_csv, read},
    write::{write_csv, replace_column},
};