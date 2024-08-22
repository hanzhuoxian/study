fn main() {
    let number_list = vec![30, 50, 25, 100, 65];
    let mut large_number = &number_list[0];

    for number in &number_list {
        if number > large_number {
            large_number = number;
        }
    }

    println!("The large number is {large_number}");

    let number_list = vec![102, 34, 87, 39, 23];

    let result = large_list(&number_list);
    println!("The large number is {result}");
}

fn large_list(list: &[i32]) -> &i32 {
    let mut large_number = &list[0];

    for number in list {
        if number > large_number {
            large_number = number;
        }
    }
    large_number
}
