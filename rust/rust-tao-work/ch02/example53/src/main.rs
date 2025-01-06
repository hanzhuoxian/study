struct Duck;

struct Pig;

// 定义 Fly trait
trait Fly {
    fn fly(&self) -> bool; // 没有函数体
}

// 为 Duck 实现 Fly trait
impl Fly for Duck {
    fn fly(&self) -> bool {
        return true;
    }
}

// 为 Pig 实现 Fly trait
impl Fly for Pig {
    fn fly(&self) -> bool {
        return false;
    }
}

//  T 代表任意类型，T: Fly 这种语法形式使用 Fly trait 对 T 进行行为上的限制
fn fly_static<T: Fly>(s: T) -> bool {
    s.fly()
}

fn fly_dyn(s: &dyn Fly) -> bool {
    s.fly()
}

fn main() {
    let pig = Pig;
    assert_eq!(fly_static::<Pig>(pig), false);
    let duck = Duck;
    assert_eq!(fly_static::<Duck>(duck), true); // ::<Duck>(duck) 用于给泛型函数指定具体的类型，在 Rust 中叫静态分发
    assert_eq!(fly_dyn(&Pig), false); // 运行时动态查找类型，动态分发。
    assert_eq!(fly_dyn(&Duck), true);
}
