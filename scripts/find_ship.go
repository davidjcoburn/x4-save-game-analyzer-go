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
	found := false
	count := 0
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.Contains(line, "id=\"[0x3b44e]\"") && strings.Contains(line, "<component") {
			found = true
			fmt.Printf("Found component at line %d\n", lineNum)
		}
		if found {
			fmt.Println(line)
			count++
			if count > 500 {
				break
			}
		}
	}
}
