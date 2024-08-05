fn main() {
    let int8: i8 = 0o17;
    let uint8: u8 = b'A'; // 单字节字符
    println!("int8 {}, uint8 {}", int8, uint8);

    let int16: i16 = 0xff;
    let uint16: u16 = 0xff;
    println!("int16 {}, uint16 {}", int16, uint16);

    let int32: i32 = 0b1111_0000;
    let uint32: u32 = 0b1111_0000;
    println!("int32 {}, uint32 {}", int32, uint32);

    let int64: i64 = 1;
    let uint64: u64 = 1;
    println!("int64 {}, uint64 {}", int64, uint64);

    // 在数字中起分隔符的作用，没有实际含义
    let int128: i128 = 1000_000;
    let uint128: u128 = 1000000;
    println!("int128 {}, uint128 {}", int128, uint128);

    // iszie 和 usize 的类型依赖运行程序的计算机架构， 64 位架构是 64 位的， 32 位架构他们是 32 位的。
    let intsize: isize = 1;
    let uintsize: usize = 1;
    println!("intsize {}, uintsize {}", intsize, uintsize);

    // 整型溢出
    let mut int8_overflow: u8 = 255;
    // println!("int8_overflow {}", int8_overflow + 1);

    int8_overflow = int8_overflow.wrapping_add(3);

    println!("int8_overflow wrapping_add {}", int8_overflow);

    // 浮点型
    let f1: f64 = 1.0;
    let f2: f32 = 3.0;
    println!("f1 {}, f2 {}", f1, f2);

    // 数学运算
    println!("5/2 = {}", 5 / 2);

    // 布尔型
    let t = true;
    let f: bool = false;
    println!("t {}, f {}", t, f);

    // 字符类型
    let c = 'z';
    let z: char = 'ℤ';
    println!("c {}, z {}", c, z);

    // 复合类型-元组（tuple）
    let tup: (i32, f64, u8) = (500, 6.4, 1);
    println!("tup.0 {}, tup.1 {}, tup.2 {}", tup.0, tup.1, tup.2)
}
