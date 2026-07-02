package main

import (
	"fmt"
	"net/http"
	"sync"
)

var count int
var mu sync.RWMutex

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/counter", counter)
	http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Path)
	fmt.Fprintf(w, "URL.PATH=%s", r.URL.Path)
	mu.Lock()
	count++
	mu.Unlock()
}

func counter(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	fmt.Println(r.URL.Path)
	fmt.Fprintf(w, "counter = %d", count)
	mu.RUnlock()
}
