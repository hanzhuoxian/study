use std::{thread, time::Duration};

#[derive(Debug, PartialEq, Copy, Clone)]
enum ShirtColor {
    Red,
    Blue,
}

struct Inventory {
    shirts: Vec<ShirtColor>,
}

impl Inventory {
    fn giveaway(&self, user_preference: Option<ShirtColor>) -> ShirtColor {
        user_preference.unwrap_or_else(||self.most_stocked_color())
    }

    fn most_stocked_color(&self) -> ShirtColor {
        let mut num_red = 0;
        let mut num_blue = 0;

        for color in &self.shirts {
            match color {
                ShirtColor::Red => num_red += 1,
                ShirtColor::Blue => num_blue += 1,
            }
        }

        if num_red >= num_blue {
            ShirtColor::Red
        } else {
            ShirtColor::Blue
        }
    }

}
fn main() {
    let store = Inventory {
        shirts: vec![ShirtColor::Blue, ShirtColor::Red, ShirtColor::Blue],
    };
    let user_pref1 = Some(ShirtColor::Red);
    let giveaway1 = store.giveaway(user_pref1);

    println!("The user with preference {:?} gets {:?}", user_pref1, giveaway1);

    let user_pref2 = None;
    let giveaway2 = store.giveaway(user_pref2);
    println!("The user with preference {:?} gets {:?}", user_pref2, giveaway2);


    let expensive_closure = |num: i32| -> i32 {
        println!("calculating slowly...");
        thread::sleep(Duration::from_secs(3));
        num
    };
    println!("expensive_closure {}", expensive_closure(1)) ;

    let add_one_v1 = |num:i32|->i32 {num+1};
    
    let list = vec![1,2,3];
    let only_borrows = || println!("From closure: {:?}", list);
    only_borrows();
    println!("From outside: {:?}", list);

    let mut list = vec![1,2,3];
    let mut borrows_mutably = || list.push(7);
    // println!("From borrows_mutably: {:?}", list); // error
    borrows_mutably();
    println!("From borrows_mutably: {:?}", list);

    let list = vec![1,2,3];

    thread::spawn(move ||println!("From thread: {:?}", list))
    .join()
    .unwrap();

    let list = vec![1,2,3];
    let fn_once = || {list};
    println!("From mut_fn: {:?}", fn_once());

}
