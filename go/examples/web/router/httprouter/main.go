package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func main() {
	r := httprouter.New()
	r.PUT("/user/installations/:installation_id/repositories/:reposit", Hello)
	r.GET("/marketplace_listing/plans/", Hello)
	r.GET("/marketplace_listing/plans/:id/accounts", Hello)
	r.GET("/search", Hello)
	r.GET("/status", Hello)
	r.GET("/support", Hello)
}

//Hello hello
func Hello(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.Write([]byte("hello"))
}
