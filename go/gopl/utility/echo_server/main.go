package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	http.HandleFunc("/", echo)
	http.ListenAndServe(":8080", nil)
}

func echo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s %s %s\n", r.Method, r.URL, r.Proto)
	fmt.Fprintf(w, "Host = %q\n", r.Host)
	fmt.Fprintf(w, "RemoteAddr: %s\n", r.RemoteAddr)
	fmt.Fprintf(w, "Header: \n")
	for h, n := range r.Header {
		fmt.Fprintf(w, "\t%s : %s\n", h, n)
	}

	if err := r.ParseForm(); err != nil {
		return
	}

	fmt.Fprintf(w, "Form:\n")
	for f, n := range r.Form {
		fmt.Fprintf(w, "\t%s : %s\n", f, n)
	}

	request, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err == nil {
		fmt.Fprintf(w, "Body:\n")
		fmt.Fprintf(w, "%s, \n", request)
	}

}
