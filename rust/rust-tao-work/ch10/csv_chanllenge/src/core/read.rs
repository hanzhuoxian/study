use crate::core::open;
use crate::err::Error;
use std::io::Read;
use std::path::PathBuf;

#[cfg(test)]
mod test {
    use std::path::PathBuf;

    use super::load_csv;

    #[test]
    fn test_valid_load_csv() {
        let filename = PathBuf::from("./input/challenge.csv");
        let csv_data = load_csv(filename);
        assert!(csv_data.is_ok());
    }
}

pub fn load_csv(csv_file: PathBuf) -> Result<String, Error> {
    let file = read(csv_file)?;
    Ok(file)
}

pub fn read(path: PathBuf) -> Result<String, Error> {
    let mut buffer = String::new();
    let mut file = open(path)?;
    let _ = file.read_to_string(&mut buffer);
    if buffer.is_empty() {
        return Err("input file missing")?;
    }
    Ok(buffer)
}
