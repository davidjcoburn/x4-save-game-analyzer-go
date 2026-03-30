// Package analyzer provides tools for analyzing X4: Foundations save games.
package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtractGameData handles the extraction of .cat files using XRCatTool.exe.
func ExtractGameData(gameRoot string, openFolder func() (string, error)) error {
	catTool := "./XRCatTool.exe"
	if _, err := os.Stat(catTool); os.IsNotExist(err) {
		fmt.Println("Error: XRCatTool.exe not found in the current directory.")
		fmt.Println("Please download 'X Catalog Tool' from Egosoft's official website:")
		fmt.Println("https://www.egosoft.com/download/x4/bonus_en.php")
		fmt.Println("(You must be logged in to your Egosoft account to see the download link)")
		fmt.Println("Alternatively, install 'X Tools' from Steam (listed under Tools in your library).")
		return fmt.Errorf("XRCatTool.exe missing")
	}

	fmt.Println("\n--- X4 Game Data Extractor ---")
	fmt.Println("This process will scan your X4 Foundations directory for .cat files")
	fmt.Println("and extract essential XML data into the './game-data' folder.")
	fmt.Println("This data is required to build the macro map for accurate ship and sector names.")
	fmt.Println()

	if gameRoot == "" {
		if openFolder != nil {
			fmt.Println("Please select your X4 Foundations root directory in the dialog...")
			p, err := openFolder()
			if err != nil && !strings.Contains(err.Error(), "not implemented") {
				return fmt.Errorf("failed to open folder dialog: %w", err)
			}
			if p != "" {
				gameRoot = p
			}
		}

		// Fallback for non-interactive, non-supported platforms, or if dialog was cancelled/not implemented
		if gameRoot == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the path to your X4 Foundations root directory: ")
			gameRoot, _ = reader.ReadString('\n')
			gameRoot = strings.TrimSpace(gameRoot)
		}
	}

	fmt.Printf("Using game root: %s\n", gameRoot)

	if _, err := os.Stat(gameRoot); os.IsNotExist(err) {
		return fmt.Errorf("game directory not found: %s", gameRoot)
	}

	outputBase := "./game-data"
	if err := os.MkdirAll(outputBase, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 1. Main game files
	fmt.Println("Processing main game files...")
	mainCats, _ := filepath.Glob(filepath.Join(gameRoot, "*.cat"))
	if len(mainCats) > 0 {
		if err := runCatTool(catTool, mainCats, filepath.Join(outputBase, "main")); err != nil {
			fmt.Printf("Warning: Failed to extract main game files: %v\n", err)
		}
	}

	// 2. Extensions
	extensionsDir := filepath.Join(gameRoot, "extensions")
	entries, err := os.ReadDir(extensionsDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				extPath := filepath.Join(extensionsDir, entry.Name())
				extCats, _ := filepath.Glob(filepath.Join(extPath, "*.cat"))
				if len(extCats) > 0 {
					fmt.Printf("Processing extension: %s\n", entry.Name())
					if err := runCatTool(catTool, extCats, filepath.Join(outputBase, entry.Name())); err != nil {
						fmt.Printf("Warning: Failed to extract extension %s: %v\n", entry.Name(), err)
					}
				}
			}
		}
	}

	fmt.Println("\nExtraction complete. You can now run the analyzer with '-build-map'.")
	return nil
}

func runCatTool(exe string, cats []string, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	args := []string{}
	for _, cat := range cats {
		args = append(args, "-in", cat)
	}
	args = append(args, "-out", outputDir, "-include", "xml")

	cmd := exec.Command(exe, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
