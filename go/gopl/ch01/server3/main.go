package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Host=%s\n", r.Host)
	fmt.Fprintf(w, "Method=%s\n", r.Method)
	fmt.Fprintf(w, "URL.PATH=%s\n", r.URL.Path)
	for h, v := range r.Header {
		for _, vv := range v {
			fmt.Fprintf(w, "Header : %s = %s\n", h, vv)
		}
	}

	if err := r.ParseForm(); err != nil {
		return
	}

	for k, v := range r.Form {
		for _, vv := range v {
			fmt.Fprintf(w, "Form : %s = %s\n", k, vv)
		}
	}

}
