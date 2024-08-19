## if let 简洁控制流

```rust
    // match 
    let config_max = Some(3u8);
    match config_max {
        Some(max) => println!("The maximum is configured to be {max}"),
        _ => (),
    }
    // if let 改写
    if let Some(max) = config_max {
        println!("The maximum is configured to be {max}");
    }
```