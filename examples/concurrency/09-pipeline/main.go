package main

import "fmt"

func generate() <-chan int {
	output := make(chan int)
	go func() {
		defer close(output)
		for i := 1; i <= 5; i++ {
			output <- i
		}
	}()
	return output
}

func square(input <-chan int) <-chan int {
	output := make(chan int)
	go func() {
		defer close(output)
		for value := range input {
			output <- value * value
		}
	}()
	return output
}

func main() {
	for value := range square(generate()) {
		fmt.Println(value)
	}
}
