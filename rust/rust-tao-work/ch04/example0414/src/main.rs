use std::vec;

fn main() {
    
    {
        let mut v = vec![1, 2, 3];
        v.push(4);
        println!("{:?}", v[1]);
    };

    // v.push(5);
}
