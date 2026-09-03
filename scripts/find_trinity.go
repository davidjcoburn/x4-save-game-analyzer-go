//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
		line := scanner.Text()
		if strings.Contains(line, "Trinity") {
			if !strings.Contains(line, "<entry") && !strings.Contains(line, "<ref") {
				fmt.Printf("Line %d: %s\n", lineNum, line)
			}
		}
	}
}
