package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	select {
	case <-time.After(2 * time.Second):
		fmt.Println("work finished")
	case <-ctx.Done():
		fmt.Println("work cancelled:", ctx.Err())
	}
}
