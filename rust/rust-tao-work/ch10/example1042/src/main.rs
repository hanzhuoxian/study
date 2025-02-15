mod read_func;

use read_func::static_kv;

fn main() {
    read_func::read_kv();
    match read_func::rw_mut_kv() {
        Ok(()) => {
            let m = static_kv::MAP_MUT
                .read()
                .map_err(|e| e.to_string())
                .unwrap();
            assert_eq!("baz", *m.get(&1).unwrap_or(&static_kv::NF));
        }
        Err(e) => {
            println!("Error {}", e)
        }
    }
}
