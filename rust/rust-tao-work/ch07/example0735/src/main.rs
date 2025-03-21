use std::process::Command;

fn main() {
    Command::new("ls")
        .arg("-l")
        .arg("-a")
        .spawn()
        .expect("ls failed");
}
