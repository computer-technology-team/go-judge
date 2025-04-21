package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Starting memory hog...")
	const mb = 1024 * 1024 // 1MB blocks
	const mbPerBlock = 10
	var data [][]byte

	for {
		block := make([]byte, mb*mbPerBlock)
		// Fill the block with data to ensure memory is actually used
		for i := 0; i < len(block); i += 1024 {
			block[i] = 1
		}
		data = append(data, block)
		fmt.Printf("Allocated %d MB, total blocks: %d\n", mbPerBlock, len(data))
		time.Sleep(50 * time.Millisecond)
	}
}
