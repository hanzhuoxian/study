package bank

import "testing"

func TestRun(t *testing.T) {
	if Balance() != 0 {
		t.Fail()
	}

	Deposit(1)

	if Balance() != 1 {
		t.Fail()
	}
}
