#[derive(Debug, PartialEq)]
struct Shoe {
    size: u32,
    style: String,
}

fn shoes_in_size(shoes: Vec<Shoe>, shoe_size: u32) -> Vec<Shoe> {
    shoes.into_iter().filter(|s| s.size == shoe_size).collect()
}

#[cfg(test)]
mod tests {
    use crate::{shoes_in_size, Shoe};

    #[test]
    fn iterator_sum() {
        let v1 = vec![1, 2, 3];
        let v1_iter = v1.iter();
        let total: i32 = v1_iter.sum();
        assert_eq!(total, 6)
    }
    #[test]
    fn iterator_map() {
        let v1 = vec![1, 3, 4];
        let v2: Vec<i32> = v1.iter().map(|x| x + 1).collect();
        assert_eq!(v2, vec![2, 4, 5])
    }
    #[test]
    fn shoe_in_size_test() {
        let shoes = vec![
            Shoe {
                size: 27,
                style: String::from("帆布鞋"),
            },
            Shoe {
                size: 28,
                style: String::from("帆布鞋"),
            },
        ];

        let except_shoes = vec![
            Shoe {
                size: 28,
                style: String::from("帆布鞋"),
            },
        ];
        let in_size = shoes_in_size(shoes, 28);
        assert_eq!(in_size, except_shoes);
    }
}
