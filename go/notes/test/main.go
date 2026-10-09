package main

import (
	"context"
	"fmt"
	"time"
)

func worker(ctx context.Context, name string, out chan<- string) error {
	tick := time.NewTicker(20 * time.Microsecond)
	defer tick.Stop()

	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			select {
			case out <- fmt.Sprintf("%s#%d", name, i):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

}
