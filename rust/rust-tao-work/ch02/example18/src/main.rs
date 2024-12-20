fn while_true(x :i32) -> i32 {
    while true {
        return x + 1;
    }

    return 1; // 去掉该行会报错，但是代码永远都走不到该行。
}
fn main() {
    let y = while_true(5);
    assert_eq!(y, 6);
}
