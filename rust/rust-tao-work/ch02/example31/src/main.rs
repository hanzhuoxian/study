fn main() {
    let mut x = 10;
    let ptr_x = &mut x as *mut i32; // 通过 as 将 &mut x引用转变为 *mut i32 可变原生指针  ptr_x

    let y = Box::new(20);
    let ptr_y = &*y as *const i32;

    unsafe {
        *ptr_x += *ptr_y
    }

    assert_eq!(x, 30)
}
