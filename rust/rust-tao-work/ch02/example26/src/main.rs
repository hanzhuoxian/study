fn main() {
    let _x = 'r';
    let _x  = 'U';
    println!("{}", '\'');
    println!("{}", '\\');
    println!("{}", '\n');
    println!("{}", '\r');
    println!("{}", '\t');
    assert_eq!('\x2A', '*');
    assert_eq!('\x25', '%');
    assert_eq!('\u{CA0}', 'ಠ');
    assert_eq!('\u{151}', 'ő');
    assert_eq!('%' as i8, 37);
    assert_eq!('ಠ' as i8, -96);

}
