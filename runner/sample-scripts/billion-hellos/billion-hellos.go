package main

import "fmt"

func main() {
	for i := 0; i < 1000000000; i++ {
		fmt.Printf("Hello %d \n", i)
	}
}
