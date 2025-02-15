use lazy_static::lazy_static;
use std::{collections::HashMap, sync::RwLock};

pub const NF: &'static str = "Not Found";

lazy_static! {

    pub static ref MAP: HashMap<u32, &'static str> = {
        let mut map = HashMap::new();
        map.insert(0, "foo");
        map
    };
    
    pub static ref MAP_MUT: RwLock<HashMap<u32, &'static str>> = {
        let mut map = HashMap::new();
        map.insert(0, "bar");
        RwLock::new(map)
    };
}
