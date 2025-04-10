
use std::{path::PathBuf, process};
use structopt::StructOpt;
use csv_chanllenge::{load_csv, replace_column, write_csv, Opt};

fn main() {
    let opt = Opt::from_args();
    let filename = PathBuf::from(opt.input);
    let csv_data = match load_csv(filename) {
        Ok(fname) => fname,
        Err(e) => {
            println!("main error : {:?}", e);
            process::exit(1)
        }
    };

    let output_file = &opt.output.unwrap_or("output/output.csv".to_string());
    let modify_data = match replace_column(csv_data, &opt.column_name, &opt.replacement) {
        Ok(data) => data,
        Err(e) => {
            println!("main error: {:?}", e);
            process::exit(1);
        }
    };

    match write_csv(&modify_data, &output_file) {
        Ok(_) => {
            println!("write success!");
        }
        Err(e) => {
            println!("main error : {:?}", e);
            process::exit(1)
        }
    }
}
