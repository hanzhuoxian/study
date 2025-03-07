struct User {
    name: &'static str,
    avatar_url: &'static str,
}
impl User {
    fn show(&self) {
        println!("user: {:?}", self.name);
        println!("avatar_url: {:?}", self.avatar_url);
    }
}
fn main() {
    let user = User {
        name: "Alex",
        avatar_url: "https://avatar.com/alex",
    };

    user.show();
    User::show(&user);
}
