package main

import (
	"fmt"
	"sync"

	"github.com/hanzhuoxian/study/go/gopl/ch09/bank"
)

func main() {
	var wg sync.WaitGroup

	wg.Go(func() {
		bank.Deposit(200)
		fmt.Println(bank.Balance())
	})

	wg.Go(func() { bank.Deposit(100) })

	wg.Wait()

	fmt.Println(bank.Balance())
}
