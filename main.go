package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: rawdog <file with interfaces> <output file for mocks>")
		return
	}
	input := os.Args[1]
	output := os.Args[2]
	makeMocks(input, output)
}
