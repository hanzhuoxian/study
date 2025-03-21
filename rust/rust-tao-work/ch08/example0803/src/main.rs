fn main() {
    let mut b = [0;3];
    let dao = '道';
    let dao_str = dao.encode_utf8(&mut b);
    assert_eq!(dao_str, "道");
    assert_eq!(3, dao_str.len());
}
