package main

import "fmt"

func main() {
	jobs := make(chan string, 2)

	jobs <- "job-1"
	jobs <- "job-2"
	close(jobs)

	for job := range jobs {
		fmt.Println(job)
	}
}
