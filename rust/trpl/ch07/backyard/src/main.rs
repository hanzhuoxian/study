use crate::garden::vegetables::Asparagus;

pub mod garden;

fn main() {
    let rose: garden::Rose = garden::Rose {};
    println!("I'm growing {rose:?}!");
    let plant = Asparagus {};
    println!("I'm growing {plant:?}!");
}
