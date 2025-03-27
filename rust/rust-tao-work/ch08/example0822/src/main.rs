fn main() {
    let v = "abcXXXabcYYYabc".matches("abc").collect::<Vec<&str>>();
    println!("{:?}", v);

    let v = "1abc2abc3".rmatches(char::is_numeric).collect::<Vec<&str>>();
    println!("{:?}", v);

    let v = "abcXXXabcYYYabc".match_indices("abc").collect::<Vec<_>>();
    println!("{:?}", v);

    let v = "abcXXXabcYYYabc".rmatch_indices("abc").collect::<Vec<_>>();
    println!("{:?}", v);

}
