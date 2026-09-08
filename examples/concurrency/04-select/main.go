package main

import (
	"fmt"
	"time"
)

func main() {
	fast := make(chan string)
	slow := make(chan string)

	go func() {
		time.Sleep(100 * time.Millisecond)
		fast <- "fast result"
	}()

	go func() {
		time.Sleep(300 * time.Millisecond)
		slow <- "slow result"
	}()

	select {
	case result := <-fast:
		fmt.Println(result)
	case result := <-slow:
		fmt.Println(result)
	case <-time.After(500 * time.Millisecond):
		fmt.Println("timed out")
	}
}
