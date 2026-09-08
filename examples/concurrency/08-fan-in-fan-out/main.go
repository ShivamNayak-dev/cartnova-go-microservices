package main

import (
	"fmt"
	"sync"
)

func source(start int) <-chan int {
	output := make(chan int)
	go func() {
		defer close(output)
		for i := 0; i < 3; i++ {
			output <- start + i
		}
	}()
	return output
}

func fanIn(inputs ...<-chan int) <-chan int {
	output := make(chan int)
	var wg sync.WaitGroup
	wg.Add(len(inputs))

	for _, input := range inputs {
		go func(channel <-chan int) {
			defer wg.Done()
			for value := range channel {
				output <- value
			}
		}(input)
	}

	go func() {
		wg.Wait()
		close(output)
	}()

	return output
}

func main() {
	results := fanIn(source(1), source(10))
	for result := range results {
		fmt.Println(result)
	}
}
