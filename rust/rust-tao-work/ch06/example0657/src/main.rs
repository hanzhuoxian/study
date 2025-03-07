fn main() {

    let v = vec![1, 2, 3, 4];
    for i in v {
        println!("{:?}", i);
    }

    // for 循环是一个语法糖
    let v = vec![1, 2, 3, 4];
    {
        let mut _iterator = v.iter();
        loop {
            match _iterator.next() {
                Some(i) => {
                    println!("{:?}", i);
                }
                None => break,
            }
        }
    }
}
