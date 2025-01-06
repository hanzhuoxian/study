#[derive(Debug, PartialEq)]
struct People {
    // 定义结构体，结构体名称遵循驼峰
    name: &'static str,
    gender: u32,
}

impl People {
    // 给结构体 People 定义方法
    fn new(name: &'static str, gender: u32) -> Self {
        return People {
            name: name,
            gender: gender,
        };
    }

    fn name(&self) {
        println!("name {:?}", self.name)
    }

    fn set_name(&mut self, name: &'static str) {
        self.name = name
    }

    fn gender(&self) {
        let gender = if self.gender == 1 { "boy" } else { "girl" };

        println!("gender : {}", gender);
    }
}

fn main() {
    let p: People = People::new("韩卓贤", 1); // 创建结构体实例
    p.name(); // 使用圆点记号来调用结构体方法
    p.gender();
    assert_eq!(
        p,
        People {
            name: "韩卓贤",
            gender: 1
        }
    );

    let mut alice = People::new("Alice", 0);
    alice.name();
    alice.gender();
    assert_eq!(
        alice,
        People {
            name: "Alice",
            gender: 0
        }
    );
    alice.set_name("Rose");
    alice.name();
    assert_eq!(
        alice,
        People {
            name: "Rose",
            gender: 0
        }
    );
}
