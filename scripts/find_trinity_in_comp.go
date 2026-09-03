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
	inComp := false
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.Contains(line, "id=\"[0x3b44e]\"") && strings.Contains(line, "<component") {
			inComp = true
			fmt.Printf("Started component at line %d\n", lineNum)
		}
		if inComp {
			if strings.Contains(line, "Trinity") {
				fmt.Printf("Found 'Trinity' at line %d: %s\n", lineNum, line)
			}
			if strings.Contains(line, "</component>") {
				// We need to track nesting to know when the ship component ends
			}
		}
	}
}
