type MathOp = fn(i32, i32) -> i32;
fn math(op: &str) -> MathOp {
    fn sum(a: i32, b: i32) -> i32 {
        a + b
    }
    fn product(a: i32, b: i32) -> i32 {
        a * b
    }
    match op {
        "sum" => sum,
        "product" => product,
        _ => {
            println!("Warning: Not implemented {:?} operator, replace with sum", op);
            sum
        }
    }
}
fn main() {
    let (a, b) = (1,2);
    let sum = math("sum");
    let product = math("product");
    let div = math("div");
    assert_eq!(sum(a,b), 3);
    assert_eq!(product(a,b), 2);
    assert_eq!(div(a,b), 3);
}
