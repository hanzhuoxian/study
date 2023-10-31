package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hanzhuoxian/study/go/gopl/ch03/pkg/surface"
)

func main() {
	http.HandleFunc("/", surfaceHandle)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func surfaceHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", surface.Surface())
}
