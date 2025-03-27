fn main() {
    let bananas = "bananas";
    assert!(bananas.contains('a'));
    assert!(bananas.contains("an"));
    assert!(bananas.contains(char::is_lowercase));
    assert!(bananas.starts_with("ba"));
    assert!(bananas.ends_with("nas"));
}
