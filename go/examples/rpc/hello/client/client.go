package main

import (
	"fmt"
	"log"
	"net/rpc"
)

func main() {
	client, err := rpc.Dial("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("dialing: ", err)
	}

	var reply string
	err = client.Call("HelloService.Hello", "hi", &reply)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(reply)

	var ad string
	err = client.Call("HelloService.Ad", "", &ad)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(ad)

}
