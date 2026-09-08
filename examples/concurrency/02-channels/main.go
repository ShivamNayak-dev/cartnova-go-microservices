package main

import "fmt"

func main() {
	messages := make(chan string)

	go func() {
		messages <- "hello from goroutine"
	}()

	message := <-messages
	fmt.Println(message)
}
