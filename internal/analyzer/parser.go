package analyzer

import (
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

//go:embed macro_map.json
var defaultMacroMap []byte

// X4SaveScanner processes X4 save game XML streams.
type X4SaveScanner struct {
	filePath string
	macroMap map[string]string
}

func NewX4SaveScanner(filePath string) *X4SaveScanner {
	s := &X4SaveScanner{
		filePath: filePath,
	}
	s.loadMacroMap()
	return s
}

func (s *X4SaveScanner) loadMacroMap() {
	s.macroMap = make(map[string]string)

	// Strategy:
	// 1. Check for local "macro_map.json" (allows user override)
	// 2. Fallback to embedded defaultMacroMap

	var data []byte
	var err error

	if _, err = os.Stat("macro_map.json"); err == nil {
		data, err = os.ReadFile("macro_map.json")
	}

	if len(data) == 0 {
		data = defaultMacroMap
	}

	if len(data) > 0 {
		var rawMap map[string]string
		if err := json.Unmarshal(data, &rawMap); err == nil {
			for k, v := range rawMap {
				s.macroMap[strings.ToLower(k)] = v
			}
		}
	}
}

func (s *X4SaveScanner) getName(macro string, attrName string) string {
	if attrName != "" && attrName != "Unnamed" {
		return attrName
	}
	if name, ok := s.macroMap[strings.ToLower(macro)]; ok {
		return name
	}
	return "Unnamed"
}

func (s *X4SaveScanner) resolveHierarchy(targetID string, idToInfo map[string]componentInfo) (string, string, Vector3) {
	sectorCode := "Unknown"
	systemName := "Unknown"
	var globalPos Vector3

	traceID := targetID
	for traceID != "" {
		info, ok := idToInfo[traceID]
		if !ok {
			break
		}

		if info.pos != nil {
			globalPos.X += info.pos.X
			globalPos.Y += info.pos.Y
			globalPos.Z += info.pos.Z
		}

		if info.class == "sector" && sectorCode == "Unknown" {
			if info.code != "" {
				sectorCode = info.code
			} else {
				sectorCode = info.macro
			}
		}

		if info.class == "cluster" {
			if name, ok := s.macroMap[strings.ToLower(info.macro)]; ok {
				systemName = name
			} else {
				systemName = info.macro
			}
		}

		traceID = info.parent
	}

	return sectorCode, systemName, globalPos
}

// Scan performs the analysis based on the criteria provided and returns the result set.
func (s *X4SaveScanner) Scan(shipQuery string, findKhaak, findUnowned, findVaults bool) (*AnalysisResults, error) {
	f, err := os.Open(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open save file: %w", err)
	}
	defer f.Close()

	var reader io.Reader = f
	if strings.HasSuffix(s.filePath, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize gzip reader: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	return s.ScanReader(reader, shipQuery, findKhaak, findUnowned, findVaults)
}

func (s *X4SaveScanner) ScanReader(reader io.Reader, shipQuery string, findKhaak, findUnowned, findVaults bool) (*AnalysisResults, error) {
	query := strings.ToLower(shipQuery)
	results := NewAnalysisResults()
	idToInfo := make(map[string]componentInfo)

	decoder := xml.NewDecoder(reader)
	parentStack := make([]string, 0, 16) // Pre-allocate small slice

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error decoding XML: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			localName := t.Name.Local
			if localName == "component" {
				var cid, class, macro, owner, name, code, state, readStatus string
				for _, attr := range t.Attr {
					switch attr.Name.Local {
					case "id":
						cid = attr.Value
					case "class":
						class = attr.Value
					case "macro":
						macro = attr.Value
					case "owner":
						owner = attr.Value
					case "name":
						name = attr.Value
					case "code":
						code = attr.Value
					case "state":
						state = attr.Value
					case "read":
						readStatus = attr.Value
					}
				}

				parentID := ""
				if len(parentStack) > 0 {
					parentID = parentStack[len(parentStack)-1]
				}

				info := componentInfo{
					parent: parentID,
					class:  class,
					macro:  macro,
					code:   code,
				}

				idToInfo[cid] = info
				parentStack = append(parentStack, cid)

				// Filter trackers
				isShip := shipClasses[class]
				isStation := stationClasses[class]
				isVault := vaultClasses[class]

				if findKhaak && owner == "khaak" && isStation {
					lm := strings.ToLower(macro)
					stype := "Installation"
					found := false
					for _, s := range khaakHiveSubstr {
						if strings.Contains(lm, s) {
							stype = "Hive"
							found = true
							break
						}
					}
					if !found {
						for _, s := range khaakNestSubstr {
							if strings.Contains(lm, s) {
								stype = "Nest"
								found = true
								break
							}
						}
					}
					if !found {
						for _, s := range khaakInstallationSubstr {
							if strings.Contains(lm, s) {
								found = true
								break
							}
						}
					}

					if found {
						results.Khaak = append(results.Khaak, ScanResult{
							ID: cid, Type: stype, Macro: macro, IsWreck: state == "wreck",
						})
					}
				}

				if query != "" && isShip {
					displayName := s.getName(macro, name)
					if strings.Contains(strings.ToLower(displayName), query) ||
						strings.Contains(strings.ToLower(macro), query) ||
						(code != "" && strings.Contains(strings.ToLower(code), query)) {
						results.Ships = append(results.Ships, ScanResult{
							ID: cid, Name: displayName, Class: class, Macro: macro,
							Owner: owner, Code: code, IsWreck: state == "wreck",
						})
					}
				}

				if findUnowned && isShip && owner == "ownerless" && state != "wreck" {
					results.Unowned = append(results.Unowned, ScanResult{
						ID: cid, Name: s.getName(macro, name), Class: class,
						Macro: macro, Owner: owner, Code: code,
					})
				}

				if findVaults && isVault {
					results.Vaults = append(results.Vaults, ScanResult{
						ID: cid, Name: "Data Vault", Macro: macro,
						IsDecrypted: readStatus == "1",
					})
				}

			} else if localName == "position" {
				if len(parentStack) > 0 {
					cid := parentStack[len(parentStack)-1]
					var pos Vector3
					for _, attr := range t.Attr {
						val, _ := strconv.ParseFloat(attr.Value, 64)
						switch attr.Name.Local {
						case "x":
							pos.X = val
						case "y":
							pos.Y = val
						case "z":
							pos.Z = val
						}
					}
					info := idToInfo[cid]
					info.pos = &pos
					idToInfo[cid] = info
				}
			} else if localName == "game" || localName == "player" || localName == "save" {
				// Capture basic save attributes
				for _, attr := range t.Attr {
					results.Info[localName+"_"+attr.Name.Local] = attr.Value
				}
			}

		case xml.EndElement:
			if t.Name.Local == "component" {
				if len(parentStack) > 0 {
					parentStack = parentStack[:len(parentStack)-1]
				}
			}
		}
	}

	// Final Resolution Step
	for i := range results.Khaak {
		results.Khaak[i].Sector, results.Khaak[i].System, results.Khaak[i].Pos = s.resolveHierarchy(results.Khaak[i].ID, idToInfo)
	}
	for i := range results.Ships {
		results.Ships[i].Sector, results.Ships[i].System, results.Ships[i].Pos = s.resolveHierarchy(results.Ships[i].ID, idToInfo)
	}
	for i := range results.Unowned {
		results.Unowned[i].Sector, results.Unowned[i].System, results.Unowned[i].Pos = s.resolveHierarchy(results.Unowned[i].ID, idToInfo)
	}
	for i := range results.Vaults {
		results.Vaults[i].Sector, results.Vaults[i].System, results.Vaults[i].Pos = s.resolveHierarchy(results.Vaults[i].ID, idToInfo)
	}

	return results, nil
}
