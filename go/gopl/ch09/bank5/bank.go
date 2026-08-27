package bank3

import "sync"

var balance int
var mu sync.RWMutex

func Deposit(amount int) {
	mu.Lock()
	balance += amount
	mu.Unlock()
}

func Withdraw(amount int) bool {
	mu.Lock()
	defer mu.Unlock()
	if balance < amount {
		return false
	}
	balance -= amount
	return true
}

func Balance() int {
	mu.RLock()
	defer mu.RUnlock()
	return balance
}
