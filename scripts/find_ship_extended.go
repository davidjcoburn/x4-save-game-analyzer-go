//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	f, err := os.Open("data/save_001.xml")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum >= 3664129 && lineNum <= 3665129 {
			fmt.Println(scanner.Text())
		}
		if lineNum > 3665129 {
			break
		}
	}
}
