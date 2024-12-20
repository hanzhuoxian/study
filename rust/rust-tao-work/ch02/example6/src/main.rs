fn main() {
    let a = [1, 2, 3]; // 定义固定长度数组
    let b = &a; // 取 a 的内存地址赋值给 b
    println!("a pointer {:p} , b pointer {:p}", &a, b);

    let mut c = vec![1, 2, 3]; // 声明动态长度数组
    println!("c pointer is {:p}", &c);
    let d = &mut c; // 获取 c 可变引用赋值给 d，要获取可变引用必须先声明可变绑定
    d.push(4);
    println!("d pointer is {:p}, d is {:?} ", d, d);

    let e = &42;
    assert_eq!(42, *e);

    // 值表达式在上下文中求值时会被创建临时值 let e = &42; 的代码演示
    let mut _0: &i32;
    let mut _1: i32;
    _1 = 42i32;
    _0 = &_1
}
