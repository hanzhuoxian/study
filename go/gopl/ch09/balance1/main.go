package main

import (
	"fmt"
	"time"

	"github.com/hanzhuoxian/study/go/gopl/ch09/bank1"
)

func main() {
	bank1.Deposit(100)
	fmt.Println(bank1.Balance())
	fmt.Println(bank1.Balance())

	time.Sleep(10 * time.Second)
}
