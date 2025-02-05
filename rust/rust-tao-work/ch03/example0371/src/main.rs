#[stable(feature = "rust1", since = "1.0.0")]
impl<T, U> Into<U> for T
where
    U: From<T>,
{
    /// Calls `U::from(self)`.
    ///
    /// That is, this conversion is whatever the implementation of
    /// <code>[From]&lt;T&gt; for U</code> chooses to do.
    #[inline]
    #[track_caller]
    fn into(self) -> U {
        U::from(self)
    }
}

fn main() {
    println!("Hello, world!");
}
