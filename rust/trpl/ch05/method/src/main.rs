#[derive(Debug)]
struct Rectangle {
    width: u32,
    height: u32,
}

impl Rectangle {
    fn area(&self) -> u32 {
        self.width * self.height
    }

    fn square(size: u32) -> Self {
        Self {
            width: size,
            height: size,
        }
    }

    fn can_hold(&self, rect: &Rectangle) -> bool {
        if self.width > rect.width && self.height > rect.height {
            return true;
        }
        false
    }
}

fn main() {
    let rect1 = Rectangle {
        width: 30,
        height: 50,
    };

    println!(
        "The area of the rectangle is {} square pixels.",
        rect1.area()
    );

    let rect2 = Rectangle {
        width: 20,
        height: 40,
    };

    println!("rect1 can hold rect2: {}", rect1.can_hold(&rect2));

    let rect3 = Rectangle {
        width: 40,
        height: 40,
    };

    let can_hold = rect1.can_hold(&rect3);
    println!("rect1 can hold rect3: {}", can_hold);

    let square = Rectangle::square(5);
    println!("{:#?}", square);
}
