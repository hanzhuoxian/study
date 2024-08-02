package foobarbaz

// Foo foo
type Foo struct {
	X int
}

// ProvideFoo returs a Foo
func ProvideFoo() Foo {
	return Foo{X: 42}
}
