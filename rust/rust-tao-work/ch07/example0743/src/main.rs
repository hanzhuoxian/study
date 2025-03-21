pub struct Letter {
    text: String,
}

pub struct EmptyEnvelope {}

pub struct ClosedEnvelope {
    letter: Letter,
}

pub struct PickupLorryHandle {
    done: bool,
}

impl Letter {
    pub fn new(text: String) -> Letter {
        Letter { text: text }
    }
}

impl EmptyEnvelope {
    pub fn wrap(&mut self, letter: Letter) -> ClosedEnvelope {
        ClosedEnvelope { letter: letter }
    }
}

pub fn buy_envelope() -> EmptyEnvelope {
    EmptyEnvelope {}
}

impl PickupLorryHandle {
    pub fn pickup(&mut self, envelope: ClosedEnvelope) {}

    pub fn done(&mut self) {
        self.done = true;
    }
}

impl Drop for PickupLorryHandle {
    fn drop(&mut self) {
        println!("sent");
    }
}

pub fn order_pickup() -> PickupLorryHandle {
    PickupLorryHandle { done: false }
}

fn main() {
    let letter = Letter::new("Hello".to_string());
    let mut envelope = buy_envelope();
    let mut pickup = order_pickup();
    let closed_envelope = envelope.wrap(letter);
    pickup.pickup(closed_envelope);
}
