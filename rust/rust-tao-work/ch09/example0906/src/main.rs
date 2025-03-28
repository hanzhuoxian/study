fn get_shortest(names: Vec<&str>) -> Option<&str> {
    if names.is_empty() {
        return None;
    }
    let mut shortest = names[0];
    for &name in names.iter() {
        if name.len() < shortest.len() {
            shortest = name;
        }
    }
    Some(shortest)
}

fn show_shortest(names: Vec<&str>) -> &str {
    // get_shortest(names).unwrap()
    // get_shortest(names).unwrap_or("Not Found")
    // get_shortest(names).unwrap_or_else(||"Not Found")
    get_shortest(names).expect("exit")

}
fn main() {
    assert_eq!(show_shortest(vec!["Uku", "Felipe"]), "Uku");
    assert_eq!(show_shortest(Vec::new()), "Not Found");
}
