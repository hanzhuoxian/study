use std::collections::{BTreeMap, HashMap};

fn main() {
    let mut hmap = HashMap::new();
    hmap.insert(3, 'c');
    hmap.insert(1, 'a');
    hmap.insert(2, 'b');
    hmap.insert(5, 'e');
    hmap.insert(4, 'd');
    println!("{:?}", hmap);

    let mut bmap = BTreeMap::new();
    bmap.insert(3, 'c');
    bmap.insert(2, 'b');
    bmap.insert(1, 'a');
    bmap.insert(5, 'e');
    bmap.insert(4, 'd');

    println!("{:?}", bmap);
}
