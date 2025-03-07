struct Counter {
    count: usize,
}

impl Iterator for Counter {
    type Item = usize;
    fn next(&mut self) -> Option<Self::Item> {
        self.incr();
        if self.count < 6 {
            Some(self.count)
        } else {
            None
        }
    }
}

trait New {
    fn new() -> Self;
}

impl New for Counter {
    fn new() -> Self {
        Self { count: 0 }
    }
}

trait Incr {
    fn incr(&mut self);
}

impl Incr for Counter {
    fn incr(&mut self) {
        self.count += 1;
    }
}

fn main() {
    let mut counter = Counter::new();
    assert_eq!(counter.next(), Some(1));
}
