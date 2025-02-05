use regex::Regex;

fn main() {
    // (?x) 忽略空格并允许行注释
    let re = Regex::new(r"(?x)(?P<year>\d{4}) #the year
    -(?P<month>\d{2}) # the month
    -(?P<day>\d{2}) # the day
    ").unwrap();

    let caps = re.captures("2018-01-01").unwrap();
    assert_eq!("2018", &caps["year"]);
    assert_eq!("01", &caps["month"]);
    assert_eq!("01", &caps["day"]);

    let after = re.replace_all("2018-01-02", "$month/$day/$year");
    assert_eq!("01/02/2018", after);
}
