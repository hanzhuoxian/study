use crate::err::Error;
use std::fs::File;
use std::io::Write;

/// #Usage:
/// ```ignore
/// let filename = PathBuf::from("./input/challenge.csv");
/// let csv_data = load_csv(filename);
/// assert!(csv_data.is_ok());
/// let csv_data = csv_data.unwrap();
/// let modify_data = replace_column(csv_data, "City", "Beijing");
/// assert!(modify_data.is_ok());
/// let modify_data = modify_data.unwrap();
/// let output_file = write_csv(&modify_data, "output/test.csv");
/// assert!(output_file.is_ok());
/// ```
pub fn write_csv(csv_data: &str, filename: &str) -> Result<(), Error> {
    write(csv_data, filename)?;
    Ok(())
}

pub fn write(data: &str, filename: &str) -> Result<(), Error> {
    let mut buffer = File::create(filename)?;
    buffer.write_all(data.as_bytes())?;
    Ok(())
}

pub fn replace_column(data: String, column: &str, replacement: &str) -> Result<String, Error> {
    let mut lines = data.lines();
    let headers = lines.next().unwrap();
    let columns: Vec<&str> = headers.split(',').collect();
    let column_number = columns.iter().position(|&e| e == column);
    let column_number = match column_number {
        Some(column) => column,
        None => Err("column name doesn't exist in the input file")?,
    };

    let mut result = String::with_capacity(data.capacity());
    result.push_str(headers);
    result.push('\n');

    for line in lines {
        let mut records: Vec<&str> = line.split(",").collect();
        records[column_number] = replacement;
        result.push_str(&records.join(","));
        result.push('\n');
    }

    Ok(result)
}
