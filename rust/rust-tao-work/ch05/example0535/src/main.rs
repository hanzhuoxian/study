struct Foo<'a>{
    part : &'a str,
}

fn main() {
    let words = String::from("Sometimes think, the greatest sorrow than older");
    let first = words.split(",").next().expect("Couldn't find a ','");
    let f = Foo{part: first};
    assert_eq!("Sometimes think", f.part);
}
