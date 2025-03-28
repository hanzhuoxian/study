use std::fs::File;
use std::io::Read;

fn main() {
    let args = std::env::args().collect::<Vec<_>>();
    println!("{:?}", args);

    if args.len() < 2 {
        println!("Usage: {} Please input file name", args[0]);
        return;
    }

    let file_name = &args[1];
    let mut file = File::open(file_name).unwrap();
    let mut contents = String::new();
    file.read_to_string(&mut contents).unwrap();
    let mut sum = 0;
    for line in contents.lines() {
        let number = line.parse::<i32>().unwrap();
        sum += number;
    }
    println!("{}", sum);
}
