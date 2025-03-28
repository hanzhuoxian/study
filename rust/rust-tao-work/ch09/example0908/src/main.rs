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

fn get_shortest_len(names: Vec<&str>) -> Option<usize> {
    get_shortest(names).map(|name| name.len())
}

fn main() {
    assert_eq!(get_shortest_len(vec!["Uku", "Felipe"]), Some(3));
    assert_eq!(get_shortest_len(Vec::new()), None);
}
