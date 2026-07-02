package main

func main() {
	fooBar, err := initializeFooBar()
	if err != nil {
		panic(err)
	}
	_ = fooBar
}
