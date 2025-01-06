#[derive(Debug, PartialEq)]
enum IpAddr {
    V4(u8, u8, u8, u8),
    V6(String),
}
fn main() {
    let x : fn(u8,u8,u8,u8) -> IpAddr = IpAddr::V4;
    let y : fn(String) -> IpAddr = IpAddr::V6;
    let home = IpAddr::V4(127, 0, 0, 1);
    assert_eq!(x(127,0,0,1), home);
    let home_v6 = IpAddr::V6(String::from("::::"));
    assert_eq!(y(String::from("::::")), home_v6);
}
