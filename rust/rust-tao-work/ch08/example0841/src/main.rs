fn main() {
    let mut vec = Vec::with_capacity(10);
    for i in 1..10 {
        vec.push(i);
    }
    vec.truncate(0);
    assert_eq!(vec, []);
    assert_eq!(vec.capacity(), 10);

    for i in 1..10 {
        vec.push(i);
    }
    vec.clear();
    assert_eq!(vec.capacity(), 10);

    vec.shrink_to_fit();
    assert_eq!(vec.capacity(), 0);
    for i in 1..100 {
        vec.push(i);
        println!("{:?}", vec.capacity());
    }


}
