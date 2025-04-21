package main

import (
	"fmt"
)

func main() {
	var a, b int
	_, err := fmt.Scan(&a, &b)
	if err != nil {
		fmt.Println("Invalid input.")
		return
	}
	fmt.Printf("%d", a+b)
}
