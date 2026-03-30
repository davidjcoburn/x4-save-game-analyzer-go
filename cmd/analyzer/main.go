// Package main is the entry point for the X4 Save Game Analyzer.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
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

	isInteractive := false
	if *path == "" {
		isInteractive = true
		p, err := openFileDialog()
		if err != nil {
			fmt.Printf("Error opening file dialog: %v\n", err)
			return
		}
		if p == "" {
			fmt.Println("No file selected. Exiting.")
			return
		}
		*path = p

		if !*khaak && !*unowned && !*vaults && *ship == "" {
			interactiveMenu(ship, khaak, unowned, vaults)
		}
	}

	if _, err := os.Stat(*path); os.IsNotExist(err) {
		log.Fatalf("Error: Save file not found at %s", *path)
	}

	fmt.Printf("\n--- Analysis Started ---\n")
	fmt.Printf("File: %s\n\n", *path)

	startTime := time.Now()
	scanner := analyzer.NewX4SaveScanner(*path)
	
	onProgress := func(percent float64) {
		fmt.Printf("\rScanning... %.0f%%", percent)
	}

	results, err := scanner.Scan(*ship, *khaak, *unowned, *vaults, onProgress)
	if err != nil {
		fmt.Println() // Newline after progress
		log.Fatalf("An error occurred during analysis: %v", err)
	}
	fmt.Printf("\rScanning... 100%%\n")
	endTime := time.Now()

	printResults(results, *ship, *khaak, *unowned, *vaults, endTime.Sub(startTime))

	if isInteractive {
		fmt.Print("\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
	}
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

func printResults(results *analyzer.AnalysisResults, shipQuery string, findKhaak, findUnowned, findVaults bool, duration time.Duration) {
	info := results.Info
	fmt.Printf("\nSave Game Information (Scanned in %.2fs):\n", duration.Seconds())
	fmt.Printf("  Player:       %s\n", info["player_name"])
	fmt.Printf("  Money:        %s\n", analyzer.FormatCredits(info["player_money"]))
	fmt.Printf("  Game Time:    %s\n", analyzer.FormatGameTime(info["game_time"]))

	saveDate := info["save_date"]
	if saveDate != "" {
		var ts int64
		fmt.Sscanf(saveDate, "%d", &ts)
		dt := time.Unix(ts, 0)
		fmt.Printf("  Save Date:    %s\n", dt.Format("2006-01-02 15:04:05"))
	}

	if shipQuery != "" {
		ships := results.Ships
		fmt.Printf("\nShip search for '%s' (%d found):\n", shipQuery, len(ships))
		sort.Slice(ships, func(i, j int) bool {
			return ships[i].Name < ships[j].Name
		})
		for _, s := range ships {
			wreckStr := ""
			if s.IsWreck {
				wreckStr = " [WRECKED]"
			}
			fmt.Printf("  - [%s] %s%s (%s) Code: %s\n", s.Owner, s.Name, wreckStr, s.Class, s.Code)
			fmt.Printf("    Sector: %s | System: %s\n", s.Sector, s.System)
			fmt.Printf("    Position: X: %s, Y: %s, Z: %s\n", analyzer.FormatCoord(s.Pos.X), analyzer.FormatCoord(s.Pos.Y), analyzer.FormatCoord(s.Pos.Z))
		}
	}

	if findUnowned {
		ships := results.Unowned
		fmt.Printf("\nAbandoned ships (%d found):\n", len(ships))
		sort.Slice(ships, func(i, j int) bool {
			if ships[i].System != ships[j].System {
				return ships[i].System < ships[j].System
			}
			return ships[i].Name < ships[j].Name
		})
		for _, s := range ships {
			fmt.Printf("  - %s (%s) Code: %s\n", s.Name, s.Class, s.Code)
			fmt.Printf("    Sector: %s | System: %s\n", s.Sector, s.System)
			fmt.Printf("    Position: X: %s, Y: %s, Z: %s\n", analyzer.FormatCoord(s.Pos.X), analyzer.FormatCoord(s.Pos.Y), analyzer.FormatCoord(s.Pos.Z))
		}
	}

	if findVaults {
		v := results.Vaults
		fmt.Printf("\nData Vaults (%d found):\n", len(v))
		sort.Slice(v, func(i, j int) bool {
			if v[i].System != v[j].System {
				return v[i].System < v[j].System
			}
			return v[i].Sector < v[j].Sector
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

	if findKhaak {
		targets := results.Khaak
		fmt.Printf("\nKha'ak Intelligence Report:\n")
		var hives, nests []analyzer.ScanResult
		for _, t := range targets {
			if t.Type == "Hive" {
				hives = append(hives, t)
			} else if t.Type == "Nest" {
				nests = append(nests, t)
			}
		}
		fmt.Printf("  Total Hives:         %d\n", len(hives))
		fmt.Printf("  Total Nests:         %d\n", len(nests))

		if len(targets) > 0 {
			fmt.Printf("\nDetected Locations (Global Sector Coordinates):\n")
			sort.Slice(targets, func(i, j int) bool {
				if targets[i].Type != targets[j].Type {
					return targets[i].Type < targets[j].Type
				}
				return targets[i].System < targets[j].System
			})
			for _, t := range targets {
				wreckStr := ""
				if t.IsWreck {
					wreckStr = " [WRECKED]"
				}
				fmt.Printf("  - [%s] %s%s | Sector: %s | System: %s\n", t.Type, t.Macro, wreckStr, t.Sector, t.System)
				fmt.Printf("    Position: X: %s, Y: %s, Z: %s\n", analyzer.FormatCoord(t.Pos.X), analyzer.FormatCoord(t.Pos.Y), analyzer.FormatCoord(t.Pos.Z))
			}
		}
	}
}

// interactiveMenu presents a CLI menu for selecting analysis criteria when no flags are provided.
func interactiveMenu(ship *string, khaak *bool, unowned *bool, vaults *bool) {
	fmt.Println("\n--- X4 Save Game Analyzer (Interactive Mode) ---")
	fmt.Println("Select search criteria (comma-separated numbers):")
	fmt.Println("1. Kha'ak Intelligence Report (Hives & Nests)")
	fmt.Println("2. Unowned (Abandoned) Ships")
	fmt.Println("3. Data Vaults (Decrypted status and locations)")
	fmt.Println("4. Search for ship by name")
	fmt.Println("5. All of the above")

	fmt.Print("\nChoice [1-5]: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		choice := scanner.Text()
		*khaak = strings.Contains(choice, "1") || strings.Contains(choice, "5")
		*unowned = strings.Contains(choice, "2") || strings.Contains(choice, "5")
		*vaults = strings.Contains(choice, "3") || strings.Contains(choice, "5")
		if strings.Contains(choice, "4") || strings.Contains(choice, "5") {
			fmt.Print("Enter ship name to search for (or leave blank): ")
			if scanner.Scan() {
				*ship = strings.TrimSpace(scanner.Text())
			}
		}
	}

	// Default to Kha'ak if nothing was selected
	if !*khaak && !*unowned && !*vaults && *ship == "" {
		*khaak = true
	}
}
