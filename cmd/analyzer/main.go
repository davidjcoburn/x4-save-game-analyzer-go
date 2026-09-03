// Package main is the entry point for the X4 Save Game Analyzer.
package main

import (
	"bufio"
	"cmp"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"x4-save-game-analyzer-go/internal/analyzer"
)

// main parses flags and executes the save game analysis.
func main() {
	path := flag.String("path", "", "Path to save file")
	ship := flag.String("ship", "", "Search for a ship by name")
	khaak := flag.Bool("khaak", false, "Scan for Kha'ak targets")
	unowned := flag.Bool("unowned", false, "Search for unowned/abandoned ships")
	vaults := flag.Bool("vaults", false, "List all Data Vaults")
	buildMap := flag.Bool("build-map", false, "Rebuild macro_map.json from game-data")
	extract := flag.Bool("extract", false, "Extract XML game data from .cat files using XRCatTool.exe")
	gameRoot := flag.String("game", "", "Path to X4 Foundations root directory (for extraction)")
	flag.Parse()

	isInteractive := false
	if *extract {
		if err := analyzer.ExtractGameData(*gameRoot, openFolderDialog); err != nil {
			log.Fatalf("Error extracting game data: %v", err)
		}
		return
	}

	if *buildMap {
		handleBuildMap()
		return
	}

	noPathSupplied := *path == ""
	if noPathSupplied {
		isInteractive = true
		p, err := openFileDialog()
		if err != nil {
			fmt.Printf("Error opening file dialog: %v\n", err)
			pauseBeforeExit()
			return
		}
		if p == "" {
			fmt.Println("No file selected. Exiting.")
			pauseBeforeExit()
			return
		}
		*path = p
	}

	shipSearchRequested := *ship != ""
	if !*khaak && !*unowned && !*vaults && *ship == "" {
		isInteractive = true
		shipSearchRequested = interactiveMenu(ship, khaak, unowned, vaults)
	}

	if _, err := os.Stat(*path); os.IsNotExist(err) {
		if noPathSupplied {
			fmt.Printf("Error: Save file not found at %s\n", *path)
			pauseBeforeExit()
			return
		}
		log.Fatalf("Error: Save file not found at %s", *path)
	}

	fmt.Printf("\n--- Analysis Started ---\n")
	fmt.Printf("File: %s\n\n", *path)

	startTime := time.Now()
	saveScanner := analyzer.NewX4SaveScanner(*path)

	onProgress := func(percent float64) {
		fmt.Printf("\rScanning... %.0f%%", percent)
	}

	results, err := saveScanner.Scan(shipSearchRequested, *khaak, *unowned, *vaults, onProgress)
	if err != nil {
		fmt.Println() // Newline after progress
		if noPathSupplied {
			fmt.Printf("An error occurred during analysis: %v\n", err)
			pauseBeforeExit()
			return
		}
		log.Fatalf("An error occurred during analysis: %v", err)
	}
	fmt.Printf("\rScanning... 100%%\n")
	endTime := time.Now()

	duration := endTime.Sub(startTime)
	printBaseInfo(results, duration)

	if *ship != "" {
		printShipSearch(results, *ship)
	}

	if *unowned {
		printUnowned(results)
	}

	if *vaults {
		printVaults(results)
	}

	if *khaak {
		printKhaak(results)
	}

	if isInteractive && shipSearchRequested {
		stdinScanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print("\nEnter ship name or ID to search for (blank to exit): ")
			if stdinScanner.Scan() {
				query := strings.TrimSpace(stdinScanner.Text())
				if query == "" {
					break
				}
				printShipSearch(results, query)
			} else {
				break
			}
		}
	}

	if noPathSupplied {
		pauseBeforeExit()
	}
}

func pauseBeforeExit() {
	fmt.Print("\nPress Enter to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func handleBuildMap() {
	if _, err := os.Stat("game-data"); os.IsNotExist(err) {
		fmt.Println("Error: 'game-data' directory not found.")
		fmt.Println("You must extract the XML game data before building the map.")
		fmt.Println("Run the analyzer with the extract flag first:")
		fmt.Println("  x4-analyzer -extract")
		return
	}

	onProgress := func(percent float64) {
		fmt.Printf("\rBuilding map... %.0f%%", percent)
	}

	macroMap, err := analyzer.BuildMacroMap("game-data", onProgress)
	if err != nil {
		fmt.Println() // Newline after progress
		log.Fatalf("Error building macro map: %v", err)
	}
	fmt.Println() // Newline after progress

	// Strategy: 
	// 1. Try to save to source location (for developers)
	// 2. Fallback to current directory (for users)
	savePath := "internal/analyzer/macro_map.json"
	isSourceLocation := true
	if _, err := os.Stat("internal/analyzer"); os.IsNotExist(err) {
		savePath = "macro_map.json"
		isSourceLocation = false
	}

	if err := analyzer.SaveMacroMap(savePath, macroMap); err != nil {
		log.Fatalf("Error saving macro map: %v", err)
	}
	fmt.Printf("Successfully rebuilt %s with %d entries.\n", savePath, len(macroMap))
	if isSourceLocation {
		fmt.Println("Rebuild the executable to embed the new map.")
	}
}

func printBaseInfo(results *analyzer.AnalysisResults, duration time.Duration) {
	info := results.Info
	fmt.Printf("\nSave Game Information (Scanned in %.2fs):\n", duration.Seconds())
	fmt.Printf("  Player:       %s\n", info["player_name"])
	fmt.Printf("  Money:        %s\n", analyzer.FormatCredits(info["player_money"]))
	fmt.Printf("  Game Time:    %s\n", analyzer.FormatGameTime(info["game_time"]))

	saveDate := info["save_date"]
	if saveDate != "" {
		if ts, err := strconv.ParseInt(saveDate, 10, 64); err == nil {
			dt := time.Unix(ts, 0)
			fmt.Printf("  Save Date:    %s\n", dt.Format("2006-01-02 15:04:05"))
		}
	}
}

func printShipSearch(results *analyzer.AnalysisResults, shipQuery string) {
	query := strings.ToLower(shipQuery)
	var matched []analyzer.ScanResult
	for _, s := range results.Ships {
		if strings.Contains(strings.ToLower(s.Name), query) ||
			strings.Contains(strings.ToLower(s.Macro), query) ||
			(s.Code != "" && strings.Contains(strings.ToLower(s.Code), query)) {
			matched = append(matched, s)
		}
	}

	fmt.Printf("\nShip search for '%s' (%d found):\n", shipQuery, len(matched))
	slices.SortFunc(matched, func(a, b analyzer.ScanResult) int {
		return cmp.Compare(a.Name, b.Name)
	})
	for _, s := range matched {
		wreckStr := ""
		if s.IsWreck {
			wreckStr = " [WRECKED]"
		}
		fmt.Printf("  - [%s] %s%s (%s) ID: %s\n", s.Owner, s.Name, wreckStr, s.Class, s.Code)
		fmt.Printf("    Sector: %s | System: %s\n", s.Sector, s.System)
		fmt.Printf("    Position: X: %s, Y: %s, Z: %s\n", analyzer.FormatCoord(s.Pos.X), analyzer.FormatCoord(s.Pos.Y), analyzer.FormatCoord(s.Pos.Z))
	}
}

func printUnowned(results *analyzer.AnalysisResults) {
	ships := results.Unowned
	fmt.Printf("\nAbandoned ships (%d found):\n", len(ships))
	slices.SortFunc(ships, func(a, b analyzer.ScanResult) int {
		if c := cmp.Compare(a.System, b.System); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})
	for _, s := range ships {
		fmt.Printf("  - %s (%s) ID: %s\n", s.Name, s.Class, s.Code)
		fmt.Printf("    Sector: %s | System: %s\n", s.Sector, s.System)
		fmt.Printf("    Position: X: %s, Y: %s, Z: %s\n", analyzer.FormatCoord(s.Pos.X), analyzer.FormatCoord(s.Pos.Y), analyzer.FormatCoord(s.Pos.Z))
	}
}

func printVaults(results *analyzer.AnalysisResults) {
	v := results.Vaults
	fmt.Printf("\nData Vaults (%d found):\n", len(v))
	slices.SortFunc(v, func(a, b analyzer.ScanResult) int {
		if c := cmp.Compare(a.System, b.System); c != 0 {
			return c
		}
		return cmp.Compare(a.Sector, b.Sector)
	})
	for _, val := range v {
		statusStr := " [LOCKED]"
		if val.IsDecrypted {
			statusStr = " [DECRYPTED]"
		}
		fmt.Printf("  - %s%s | Sector: %s | System: %s\n", val.Name, statusStr, val.Sector, val.System)
		fmt.Printf("    Position: X: %s, Y: %s, Z: %s\n", analyzer.FormatCoord(val.Pos.X), analyzer.FormatCoord(val.Pos.Y), analyzer.FormatCoord(val.Pos.Z))
	}
}

func printKhaak(results *analyzer.AnalysisResults) {
	targets := results.Khaak
	fmt.Printf("\nKha'ak Intelligence Report:\n")
	var hives, nests, installations []analyzer.ScanResult
	for _, t := range targets {
		switch t.Type {
		case "Hive":
			hives = append(hives, t)
		case "Nest":
			nests = append(nests, t)
		default:
			installations = append(installations, t)
		}
	}
	fmt.Printf("  Total Hives:         %d\n", len(hives))
	fmt.Printf("  Total Nests:         %d\n", len(nests))
	if len(installations) > 0 {
		fmt.Printf("  Other Installations: %d\n", len(installations))
	}
	fmt.Printf("  Total Detected:      %d\n", len(targets))

	if len(targets) > 0 {
		fmt.Printf("\nDetected Locations (Global Sector Coordinates):\n")
		slices.SortFunc(targets, func(a, b analyzer.ScanResult) int {
			if c := cmp.Compare(a.Type, b.Type); c != 0 {
				return c
			}
			return cmp.Compare(a.System, b.System)
		})
		for _, t := range targets {
			wreckStr := ""
			if t.IsWreck {
				wreckStr = " [WRECKED]"
			}
			fmt.Printf("  - [%s] %s%s | Sector: %s | System: %s\n", t.Type, t.Name, wreckStr, t.Sector, t.System)
			fmt.Printf("    Position: X: %s, Y: %s, Z: %s\n", analyzer.FormatCoord(t.Pos.X), analyzer.FormatCoord(t.Pos.Y), analyzer.FormatCoord(t.Pos.Z))
		}
	}
}

// interactiveMenu presents a CLI menu for selecting analysis criteria when no flags are provided.
// Returns true if ship search was selected.
func interactiveMenu(ship *string, khaak *bool, unowned *bool, vaults *bool) bool {
	fmt.Println("\n--- X4 Save Game Analyzer (Interactive Mode) ---")
	fmt.Println("Select search criteria (comma-separated numbers):")
	fmt.Println("1. Kha'ak Intelligence Report (Hives & Nests)")
	fmt.Println("2. Unowned (Abandoned) Ships")
	fmt.Println("3. Data Vaults (Decrypted status and locations)")
	fmt.Println("4. Search for ship by name or ID")
	fmt.Println("5. All of the above")

	fmt.Print("\nChoice [1-5]: ")
	menuScanner := bufio.NewScanner(os.Stdin)
	shipRequested := false
	if menuScanner.Scan() {
		choice := menuScanner.Text()
		*khaak = strings.Contains(choice, "1") || strings.Contains(choice, "5")
		*unowned = strings.Contains(choice, "2") || strings.Contains(choice, "5")
		*vaults = strings.Contains(choice, "3") || strings.Contains(choice, "5")
		if strings.Contains(choice, "4") || strings.Contains(choice, "5") {
			shipRequested = true
			fmt.Print("Enter ship name or ID to search for (or leave blank): ")
			if menuScanner.Scan() {
				*ship = strings.TrimSpace(menuScanner.Text())
			}
		}
	}

	// Default to Kha'ak if nothing was selected
	if !*khaak && !*unowned && !*vaults && *ship == "" {
		*khaak = true
	}
	return shipRequested
}
