package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
)

func main() {
	var ages map[string]int
	ages = make(map[string]int)
	ages["name"] = 1

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(ages); err != nil {
		log.Fatal(err)
	}

	fmt.Println(buf.String())

	var age1 map[string]int
	if err := json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&age1); err != nil {
		log.Fatal(err)
	}
	fmt.Println(age1)
}
