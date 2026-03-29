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

func main() {
	path := flag.String("path", "", "Path to save file")
	ship := flag.String("ship", "", "Search for a ship by name")
	khaak := flag.Bool("khaak", false, "Scan for Kha'ak targets")
	unowned := flag.Bool("unowned", false, "Search for unowned/abandoned ships")
	vaults := flag.Bool("vaults", false, "List all Data Vaults")
	buildMap := flag.Bool("build-map", false, "Rebuild macro_map.json from game-data")
	flag.Parse()

	if *buildMap {
		macroMap, err := analyzer.BuildMacroMap("game-data")
		if err != nil {
			log.Fatalf("Error building macro map: %v", err)
		}
		// Save to the source location so it's included in next build
		if err := analyzer.SaveMacroMap("internal/analyzer/macro_map.json", macroMap); err != nil {
			log.Fatalf("Error saving macro map: %v", err)
		}
		fmt.Printf("Successfully rebuilt internal/analyzer/macro_map.json with %d entries.\n", len(macroMap))
		fmt.Println("Rebuild the executable to embed the new map.")
		return
	}

	// If no path is supplied, try to open a file dialog
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

		// If no other flags were set, enter interactive criteria selection
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
	results, err := scanner.Scan(*ship, *khaak, *unowned, *vaults)
	if err != nil {
		log.Fatalf("An error occurred during analysis: %v", err)
	}
	endTime := time.Now()

	info := results.Info
	fmt.Printf("\nSave Game Information (Scanned in %.2fs):\n", endTime.Sub(startTime).Seconds())
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

	if *ship != "" {
		ships := results.Ships
		fmt.Printf("\nShip search for '%s' (%d found):\n", *ship, len(ships))
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

	if *unowned {
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

	if *vaults {
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

	if *khaak {
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

	if isInteractive {
		fmt.Print("\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
	}
}

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
