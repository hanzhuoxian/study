package main

import "fmt"

// **练习 9.1：** 给gopl.io/ch9/bank1程序添加一个Withdraw(amount int)取款函数。其返回结果应该要表明事务是成功了还是因为没有足够资金失败了。这条消息会被发送给monitor的goroutine，且消息需要包含取款的额度和一个新的channel，这个新channel会被monitor goroutine来把boolean结果发回给Withdraw。

// withdrawal 是发送给 teller goroutine 的取款请求：
// amount 是取款额度，reply 用于接收本次事务是否成功。
type withdrawal struct {
	amount int
	reply  chan bool
}

var deposits = make(chan int)         // send amount to deposit
var balances = make(chan int)         // receive balance
var withdraws = make(chan withdrawal) // send withdrawal request

func Deposit(amount int) {
	deposits <- amount
}

func Balance() int {
	return <-balances
}

// Withdraw 取款 amount，余额不足时返回 false 且不修改余额。
func Withdraw(amount int) bool {
	reply := make(chan bool)
	withdraws <- withdrawal{amount: amount, reply: reply}
	return <-reply
}

func teller() {
	var balance int
	for {
		select {
		case amount := <-deposits:
			balance += amount
			fmt.Println("deposit", amount, "balance", balance)
		case balances <- balance:
		case w := <-withdraws:
			if w.amount > balance {
				fmt.Println("withdraw", w.amount, "failed, balance", balance)
				w.reply <- false
				continue
			}
			balance -= w.amount
			fmt.Println("withdraw", w.amount, "balance", balance)
			w.reply <- true
		}
	}
}

func init() {
	go teller()
}

func main() {
	Deposit(200)
	fmt.Println("balance =", Balance())

	fmt.Println("withdraw 100 ->", Withdraw(100)) // true
	fmt.Println("withdraw 500 ->", Withdraw(500)) // false, 资金不足
	fmt.Println("balance =", Balance())
}
