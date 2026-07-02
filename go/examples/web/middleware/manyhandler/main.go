// middleware/hello_with_more_routes.go
// 省略了一些相同的代码
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func helloHandler(wr http.ResponseWriter, r *http.Request) {
	// ...
}

func showInfoHandler(wr http.ResponseWriter, r *http.Request) {
	// ...
}

func showEmailHandler(wr http.ResponseWriter, r *http.Request) {
	// ...
}

func showFriendsHandler(wr http.ResponseWriter, r *http.Request) {
	timeStart := time.Now()
	wr.Write([]byte("your friends is tom and alex"))
	timeElapsed := time.Since(timeStart)
	log.Println(timeElapsed)
}

func main() {
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/info/show", showInfoHandler)
	http.HandleFunc("/email/show", showEmailHandler)
	http.HandleFunc("/friends/show", showFriendsHandler)
	// ...

	err := http.ListenAndServe(":8080", nil)

	fmt.Println(err)
}
