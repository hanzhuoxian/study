use std::fmt::Debug;

pub trait Fly {
    fn fly(&self) -> bool;
}

#[derive(Debug)]
struct Duck {}

#[derive(Debug)]
struct Pig {}

impl Fly for Duck {
    fn fly(&self) -> bool {
        return true;
    }
}

impl Fly for Pig {
    fn fly(&self) -> bool {
        return false;
    }
}

fn static_fly<T: Fly>(s: T) -> bool {
    s.fly()
}

// impl Fly 方式 等价于 static_fly 中使用泛型限定
fn fly_static(s: impl Fly + Debug) -> bool {
    s.fly()
}

fn dyn_fly(s: &dyn Fly) {
    s.fly();
}

// impl Fly 方式返回值时 实际上相当于给返回类型增加一种 trait 限定范围
fn can_fly(s: impl Fly + Debug) -> impl Fly {
    if s.fly() {
        println!("{:?} can fly", s);
    } else {
        println!("{:?} can't fly", s);
    }
    s
}

// 返回 dyn trait 对象
fn dyn_can_fly(s: impl Fly+Debug+'static) -> Box<dyn Fly> {
    if s.fly() {
        println!("{:?} dyn can fly", s);
    } else {
        println!("{:?} dyn can't fly", s);
    } 
    Box::new(s)
}

fn main() {
    let pig = Pig {};
    assert_eq!(fly_static(pig), false);

    let pig = Pig {};
    assert_eq!(static_fly::<Pig>(pig), false);

    let pig = Pig {};
    dyn_fly(&pig);

    let duck = Duck {};
    assert_eq!(fly_static(duck), true);

    let pig = Pig {};
    let pig = can_fly(pig);
    pig.fly();

    let duck = Duck {};
    let duck = can_fly(duck);
    duck.fly();

    let duck = Duck{};
    let d = dyn_can_fly(duck);
    d.fly();
}
