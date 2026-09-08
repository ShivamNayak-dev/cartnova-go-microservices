package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	go printMessage("first", &wg)
	go printMessage("second", &wg)

	wg.Wait()
}

func printMessage(message string, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println(message)
}
