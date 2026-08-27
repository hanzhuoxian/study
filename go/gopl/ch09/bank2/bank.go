package bank2

var balance int
var semaphore = make(chan struct{}, 1)

func Deposit(amount int) {
	semaphore <- struct{}{}
	balance += amount
	<-semaphore
}

func Balance() int {
	semaphore <- struct{}{}
	b := balance
	<-semaphore
	return b
}
