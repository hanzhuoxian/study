#![feature(unboxed_closures, fn_traits)]
struct Closure<'a> {
    env_val: &'a u32,
}

impl<'a> FnOnce<()> for Closure<'a> {
    type Output = ();
    extern "rust-call" fn call_once(self, args: ()) {
        println!("{:?}", self.env_val)
    }
}

impl<'a> FnMut<()> for Closure<'a> {
    extern "rust-call" fn call_mut(&mut self, args: ()) {
        println!("{:?}", self.env_val);
    }
}

impl<'a> Fn<()> for Closure<'a> {
    extern "rust-call" fn call(&self, args: ()) {
        println!("{:?}", self.env_val);
    }
}

fn main() {
    let env_val = 42;
    let mut c = Closure { env_val: &env_val };
    c();
    c.call_mut(());
    c.call_once(());
    println!("Hello, world!");
}
