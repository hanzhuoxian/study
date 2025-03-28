fn double(value: f64) -> f64 {
    value * 2.0
}

fn square(value: f64) -> f64 {
    value.powi(2)
}

fn inverse(value: f64) -> f64 {
    value * -1.
}

fn log(value: f64) -> Option<f64> {
    match value.log2() {
        x if x.is_normal() => Some(x),
        _ => None,
    }
}

fn sqrt(value: f64) -> Option<f64> {
    match value.sqrt() {
        x if x.is_normal() => Some(x),
        _ => None,
    }
}

fn main() {
    let number: f64 = 20.0;
    let result = Option::from(number)
        .map(inverse)
        .map(double)
        .map(inverse)
        .and_then(log)
        .map(square)
        .and_then(sqrt);

    match result {
        Some(x) => println!("{:?}", x),
        None => println!("This failed"),
    }
}
