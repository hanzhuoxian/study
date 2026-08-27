package bank1

import "fmt"

var deposits = make(chan int) // send amount to deposit
var balances = make(chan int) // receive balance

func Deposit(amount int) {
	deposits <- amount
}

func Balance() int {
	return <-balances
}

func teller() {
	var balance int
	for {
		select {
		case amount := <-deposits:
			fmt.Println("deposit", amount)
			balance += amount
		case balances <- balance:
			fmt.Println("get balance", balance)
		}
	}
}

func init() {
	go teller()
}
