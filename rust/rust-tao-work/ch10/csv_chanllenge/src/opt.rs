use structopt_derive::*;

#[derive(Debug, StructOpt)]
#[structopt(name = "csv_chanllenge", about = "Usage")]
pub struct Opt {
    #[structopt(help = "Input file")]
    pub input: String,
    #[structopt(help = "Column Name")]
    pub column_name: String,
    #[structopt(help = "Replacement Column Name")]
    pub replacement: String,
    #[structopt(help = "Output file, stdout if not present")]
    pub output: Option<String>,
}
