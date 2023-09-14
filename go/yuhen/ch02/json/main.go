package main

import (
	"encoding/json"
	"fmt"
)

type People struct {
	Name      string
	RequestId string `json:"request_id"`
}

func main() {
	p := People{}
	json.Unmarshal([]byte(`{"name":"韩桌贤", "request_id":"xxx"}`), &p)
	fmt.Println(p)
}
