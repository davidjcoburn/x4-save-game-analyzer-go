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
		}
		if inComp {
			if strings.Contains(line, "name=") {
				fmt.Printf("Line %d: %s\n", lineNum, line)
			}
			// Look for the end of the ship component
			if strings.Contains(line, "</component>") {
				// Simplified end check
			}
		}
	}
}
