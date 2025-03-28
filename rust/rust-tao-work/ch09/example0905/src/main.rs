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
    match get_shortest(names) {
        Some(name) => name,
        None => "Not Found",
    }
}
fn main() {
    assert_eq!(show_shortest(vec!["Uku", "Felipe"]), "Uku");
    assert_eq!(show_shortest(Vec::new()), "Not Found");
}
