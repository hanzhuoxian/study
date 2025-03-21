
fn main() {
    let dao = '道';
    let dao_u32 = dao as u32;
    assert_eq!(36947, dao_u32);
    println!("{}", dao.escape_unicode());
    assert_eq!(char::from(65), 'A');
    assert_eq!(char::from_u32(0x9053), Some(dao));
    assert_eq!(char::from_u32(36947), Some(dao));
    assert_eq!(char::from_u32(4294967295), None);

}
