package foobarbaz

// Bar bar
type Bar struct {
	X int
}

// ProvideBar returns a Bar: a negative Foo
func ProvideBar(foo Foo) Bar {
	return Bar{X: -foo.X}
}
