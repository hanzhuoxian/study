use std::env;

#[derive(Debug, Clone)]
// 信件
pub struct Letter {
    text: String,
}

// 信封
pub struct Envelope {
    letter: Option<Letter>,
}

// 装车
pub struct PickupLorryHandle{
    done: bool,
}


impl  Letter {
    pub fn new(text: String) -> Letter {
        Letter {
            text: text,
        }
    }
}

impl Envelope {
    pub fn wrap(&mut self, letter: Letter) {
        self.letter = Some(letter);
    }
}

pub fn buy_envelope() -> Envelope {
    Envelope {
        letter: None,
    }
}

impl PickupLorryHandle {
    pub fn pickup(&mut self, envelope: Envelope){

    }

    pub fn done(&mut self) {
        self.done = true;
        println!("done");
    }
}

pub fn order_pickup() -> PickupLorryHandle {
    PickupLorryHandle {
        done: false,
    }
}

fn main() {
    let letter = Letter::new("Dear Rustfeat".to_string());
    let mut envelope = buy_envelope();
    envelope.wrap(letter);
    let mut lorry = order_pickup();
    lorry.pickup(envelope);
    lorry.done();
}
