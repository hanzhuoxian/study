// middleware/hello.go
package main

import (
	"fmt"
	"net/http"
)

func hello(wr http.ResponseWriter, r *http.Request) {
	wr.Write([]byte("hello"))
}

func main() {
	http.HandleFunc("/", hello)
	err := http.ListenAndServe(":8080", nil)

	fmt.Println(err)

}
