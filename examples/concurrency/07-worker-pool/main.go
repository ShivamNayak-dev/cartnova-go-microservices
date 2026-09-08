package main

import (
	"fmt"
	"sync"
)

func worker(id int, jobs <-chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		fmt.Printf("worker %d processed job %d\n", id, job)
	}
}

func main() {
	jobs := make(chan int, 10)
	var wg sync.WaitGroup

	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go worker(i, jobs, &wg)
	}

	for job := 1; job <= 10; job++ {
		jobs <- job
	}
	close(jobs)

	wg.Wait()
}
