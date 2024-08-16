#[derive(Debug)]
struct Rectangle {
    width: u32,
    height: u32,
}

fn main() {
    rectangles_v1();

    // 使用元组重构
    rectangles_v2();

    // 使用结构体重构
    rectangles_v3();
}

fn rectangles_v1() {
    let width1 = 30;
    let height1 = 50;

    println!(
        "v1: The area of the rectangle is {} square pixels.",
        area_v1(width1, height1)
    );
}

fn area_v1(width: u32, height: u32) -> u32 {
    width * height
}

fn rectangles_v2() {
    let rect1 = (30, 50);

    println!(
        "v2: The area of the rectangle is {} square pixels.",
        area_v2(rect1)
    );
}

fn area_v2(rect: (u32, u32)) -> u32 {
    rect.0 * rect.1
}

fn rectangles_v3() {
    let rect1 = Rectangle {
        width: 30,
        height: 50,
    };

    println!("rect1(debug): {:#?}", rect1);
    println!(
        "v3: The area of the rectangle is {} square pixels.",
        area_v3(&rect1)
    );

    dbg!(rect1);
}

fn area_v3(rect: &Rectangle) -> u32 {
    rect.width * rect.height
}
