package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	if err := waitForServer("http://localhost:8080"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Server is responding")
}

func waitForServer(url string) error {
	const timeout = 1 * time.Minute
	deadline := time.Now().Add(timeout)
	for tries := 0; time.Now().Before(deadline); tries++ {
		_, err := http.Get(url)
		if err == nil {
			return nil // success
		}
		log.Printf("server not responding (%s); retrying...", err)
		time.Sleep(time.Second << tries)
	}
	return fmt.Errorf("server %s failed to respond after %s", url, timeout)
}
