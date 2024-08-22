// 结构体中的泛型
#[derive(Debug)]
struct Point<T, U> {
    x: T,
    y: U,
}

// 方法重定义泛型
impl<T, U> Point<T, U> {
    fn x(&self) -> &T {
        &self.x
    }

    fn y(&self) -> &U {
        &self.y
    }
}

// 枚举中的泛型
enum Option<T> {
    Some(T),
    None,
}

fn main() {
    let binding = vec![1, 2, 3, 4];
    let max = largest_i32(&binding);
    println!("The largest number is : {}", max);
    let chars = vec!['a', 'b', 'c'];
    let max = largest_char(&chars);
    println!("The largest char is : {}", max);

    let max = largest(&binding);
    println!("The largest number is : {}", max);
    let max = largest(&chars);
    println!("The largest char is : {}", max);

    let integer = Point { x: 10, y: 10 };
    let float = Point { x: 10, y: 1.0 };
    println!("{integer:?}");
    println!("{float:?}");
    println!("integer x {}, float y {}", float.x(), float.y());
}

// 泛型定义
fn largest<T: std::cmp::PartialOrd>(list: &[T]) -> &T {
    let mut largest = &list[0];
    for item in list {
        if item > largest {
            largest = item
        }
    }
    largest
}

fn largest_i32(list: &[i32]) -> &i32 {
    let mut largest = &list[0];
    for number in list {
        if number > largest {
            largest = number
        }
    }
    largest
}

fn largest_char(list: &[char]) -> &char {
    let mut largest = &list[0];
    for number in list {
        if number > largest {
            largest = number
        }
    }
    largest
}
