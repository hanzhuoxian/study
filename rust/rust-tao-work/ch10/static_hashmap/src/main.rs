mod static_kv {
    use std::{collections::HashMap, sync::RwLock};

    use lazy_static::lazy_static;

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
}

fn read_kv() {
    let ref m = static_kv::MAP;
    assert_eq!("foo", *m.get(&0).unwrap_or(&static_kv::NF));
    assert_eq!(static_kv::NF, *m.get(&1).unwrap_or(&static_kv::NF))
}
fn rw_mut_kv() -> Result<(), String> {
    {
        let m = static_kv::MAP_MUT.read().map_err(|e| e.to_string())?;
        assert_eq!("bar", *m.get(&0).unwrap_or(&static_kv::NF));
    }

    {
        let mut m = static_kv::MAP_MUT.write().map_err(|e| e.to_string())?;
        m.insert(1, "baz");
    }

    Ok(())
}

fn main() {
    read_kv();
    match rw_mut_kv() {
        Ok(()) => {
            let m = static_kv::MAP_MUT
                .read()
                .map_err(|e| e.to_string())
                .unwrap();
            assert_eq!("baz", *m.get(&1).unwrap_or(&static_kv::NF));
        }
        Err(e) => {
            println!("Error {}", e)
        },
    }
}
