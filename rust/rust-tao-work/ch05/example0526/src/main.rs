fn compute(input: &u32, output: &mut u32) {
    let cache_input = *input;
    if cache_input > 10 {
        *output = 2;
    }else if cache_input > 5 {
        *output *= 2;
    }

    if *input > 5 {
        *output *= 2;
    }
}

fn main() {
    let i = 20;
    let mut o = 5;
    compute(&i, &mut o);
    assert_eq!(o, 4)
}
