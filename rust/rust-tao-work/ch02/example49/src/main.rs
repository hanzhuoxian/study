use std::collections::BinaryHeap;

fn main() {
    let mut heap = BinaryHeap::new();
    assert_eq!(heap.peek(), None);
    let arr = [93,80,48,53,72,30,18,36,15,35,45];
    for &i in arr.iter(){
        heap.push(i);
    }
    assert_eq!(heap.peek(), Some(&93));
    
    println!("{:?}", heap);

    println!("{:?}", heap.pop());
    println!("{:?}", heap.pop());
    println!("{:?}", heap.pop());
}
