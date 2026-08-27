package main

import "fmt"

func main() {
    numbers := []int{2, 4, 6, 8, 10}
    results := make(chan int, len(numbers))

    for _, n := range numbers {
    go func(num int) {
        results <- square(num)
    	}(n)
	}

    for range numbers {
        fmt.Println(<-results)
    }
}

func square(n int) int {
    return n * n
}
