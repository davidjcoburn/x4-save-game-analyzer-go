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
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.Contains(line, "id=\"[0x3b44e]\"") && strings.Contains(line, "<component") {
			found = true
		}
		if found {
			if strings.Contains(line, "87557") {
				fmt.Printf("Found 87557 at line %d: %s\n", lineNum, line)
			}
			if strings.Contains(line, "</component>") {
				// This might be a nested component end, but let's see
			}
		}
	}
}
