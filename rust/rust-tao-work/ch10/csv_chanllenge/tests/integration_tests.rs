#[cfg(test)]
mod test {
    use std::path::PathBuf;

    use csv_chanllenge::{load_csv, replace_column, write_csv};

    #[test]
    fn test_csv_challenge(){
        let filename = PathBuf::from("./input/challenge.csv");
        let csv_data = load_csv(filename);
        assert!(csv_data.is_ok());
        let csv_data = csv_data.unwrap();
        let modify_data = replace_column(csv_data, "City", "Beijing");
        assert!(modify_data.is_ok());
        let modify_data = modify_data.unwrap();
        let output_file = write_csv(&modify_data, "output/test.csv");
        assert!(output_file.is_ok());
    }
}