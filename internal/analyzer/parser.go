// Package analyzer provides tools for analyzing X4: Foundations save games.
package analyzer

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/klauspost/compress/gzip"
)

func isShipClass(c string) bool {
	switch c {
	case "ship", "ship_s", "ship_m", "ship_l", "ship_xl":
		return true
	}
	return false
}

func isStationClass(c string) bool {
	switch c {
	case "station", "weaponplatform":
		return true
	}
	return false
}

func isVaultClass(c string) bool {
	return c == "datavault"
}

func isModuleClass(c string) bool {
	switch c {
	case "module", "production", "storage", "dockarea", "defence":
		return true
	}
	return false
}

func isHierarchyClass(c string) bool {
	switch c {
	case "galaxy", "cluster", "sector", "station", "weaponplatform", "datavault",
		"ship", "ship_s", "ship_m", "ship_l", "ship_xl",
		"module", "production", "storage", "dockarea", "defence", "highway", "":
		return true
	}
	return false
}

//go:embed macro_map.json
var defaultMacroMap []byte

// X4SaveScanner processes X4 save game XML streams to extract specific information.
type X4SaveScanner struct {
	filePath string
	macroMap map[string]string
}

// NewX4SaveScanner creates a new X4SaveScanner for the given file path and initializes the macro map.
func NewX4SaveScanner(filePath string) *X4SaveScanner {
	s := &X4SaveScanner{
		filePath: filePath,
	}
	s.loadMacroMap()
	return s
}

// loadMacroMap loads the macro name mapping from a local JSON file or embedded defaults.
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

// getName resolves a macro and attribute name to a human-readable name.
func (s *X4SaveScanner) getName(macro string, attrName string) string {
	if attrName != "" && attrName != "Unnamed" {
		return attrName
	}
	if name, ok := s.macroMap[strings.ToLower(macro)]; ok {
		return name
	}
	if macro != "" {
		return macro
	}
	return "Unnamed"
}

// resolveHierarchy traces the component hierarchy to find sector, system, and global position.
func (s *X4SaveScanner) resolveHierarchy(targetID string, idToInfo map[string]componentInfo) (string, string, Vector3) {
	sectorCode := "Unknown"
	systemName := "Unknown"
	var sectorPos Vector3
	foundSector := false

	traceID := targetID
	depth := 0
	for traceID != "" && depth < 64 {
		depth++
		info, ok := idToInfo[traceID]
		if !ok {
			break
		}

		if info.class == "sector" && sectorCode == "Unknown" {
			// Try to get friendly name from macro first
			if name, ok := s.macroMap[strings.ToLower(info.macro)]; ok {
				sectorCode = name
			} else if info.code != "" {
				sectorCode = info.code
			} else {
				sectorCode = info.macro
			}
			foundSector = true
			traceID = info.parent
			continue // Don't add the sector's own position to the sector-relative sum
		}

		if !foundSector && info.hasPos {
			sectorPos.X += info.pos.X
			sectorPos.Y += info.pos.Y
			sectorPos.Z += info.pos.Z
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

	return sectorCode, systemName, sectorPos
}

// Scan performs the analysis on the file at s.filePath based on the provided criteria.
func (s *X4SaveScanner) Scan(findShips, findKhaak, findUnowned, findVault bool, onProgress func(float64)) (*AnalysisResults, error) {
	fileInfo, err := os.Stat(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	totalSize := fileInfo.Size()

	f, err := os.Open(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open save file: %w", err)
	}
	defer f.Close()

	var reader io.Reader = f
	if onProgress != nil {
		reader = NewProgressReader(f, totalSize, onProgress)
	}

	bufferedCompressed := bufio.NewReaderSize(reader, 512*1024)

	if strings.HasSuffix(s.filePath, ".gz") {
		gz, err := gzip.NewReader(bufferedCompressed)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize gzip reader: %w", err)
		}
		defer gz.Close()
		reader = gz
	} else {
		reader = bufferedCompressed
	}

	return s.ScanReader(reader, findShips, findKhaak, findUnowned, findVault)
}

// lastWhitespaceIndex returns the last index of whitespace (' ', '\t', '\r', '\n') in b.
func lastWhitespaceIndex(b []byte) int {
	for i := len(b) - 1; i >= 0; i-- {
		switch b[i] {
		case ' ', '\t', '\r', '\n':
			return i
		}
	}
	return -1
}

// parseAttrs iterates over the attributes and calls the callback for each key-value pair without heap-allocating keys.
func parseAttrs(attrs []byte, cb func(k, v []byte)) {
	for len(attrs) > 0 {
		eqIdx := bytes.IndexByte(attrs, '=')
		if eqIdx == -1 {
			break
		}

		keyBytes := bytes.TrimSpace(attrs[:eqIdx])
		if wsIdx := lastWhitespaceIndex(keyBytes); wsIdx != -1 {
			keyBytes = keyBytes[wsIdx+1:]
		}

		attrs = attrs[eqIdx+1:]

		quoteIdx := -1
		var quote byte
		for i, b := range attrs {
			if b == '"' || b == '\'' {
				quoteIdx = i
				quote = b
				break
			}
		}

		if quoteIdx == -1 {
			break
		}

		attrs = attrs[quoteIdx+1:]
		endQuoteIdx := bytes.IndexByte(attrs, quote)
		if endQuoteIdx == -1 {
			break
		}

		cb(keyBytes, attrs[:endQuoteIdx])
		attrs = attrs[endQuoteIdx+1:]
	}
}

var (
	bID     = []byte("id")
	bClass  = []byte("class")
	bMacro  = []byte("macro")
	bOwner  = []byte("owner")
	bName   = []byte("name")
	bCode   = []byte("code")
	bState  = []byte("state")
	bRead   = []byte("read")
	bX      = []byte("x")
	bY      = []byte("y")
	bZ      = []byte("z")
)

// ScanReader performs the analysis on an io.Reader stream based on the provided criteria.
func (s *X4SaveScanner) ScanReader(reader io.Reader, findShips, findKhaak, findUnowned, findVault bool) (*AnalysisResults, error) {
	results := NewAnalysisResults()
	idToInfo := make(map[string]componentInfo, 131072)

	parentStack := make([]string, 0, 16) // Stack of component IDs
	isCompStack := make([]bool, 0, 16)   // true if component, false if connection
	pendingPos := Vector3{}

	br := bufio.NewReaderSize(reader, 1024*1024) // 1MB read buffer

	var tComponent = []byte("component")
	var tConnection = []byte("connection")
	var tPosition = []byte("position")
	var tGame = []byte("game")
	var tPlayer = []byte("player")
	var tSave = []byte("save")

	for {
		tagData, err := br.ReadSlice('>')
		if err != nil {
			if err == io.EOF {
				break
			}
			// If buffer fills before '>' (extremely rare), fall back to ReadBytes
			if err == bufio.ErrBufferFull {
				br.ReadBytes('>')
				continue
			}
			return nil, fmt.Errorf("error reading xml stream: %w", err)
		}

		startIdx := bytes.IndexByte(tagData, '<')
		if startIdx == -1 {
			continue
		}

		inner := tagData[startIdx+1 : len(tagData)-1]
		if len(inner) == 0 {
			continue
		}

		isEnd := inner[0] == '/'
		if isEnd {
			inner = inner[1:]
		}

		isSelfClosing := false
		if len(inner) > 0 && inner[len(inner)-1] == '/' {
			isSelfClosing = true
			inner = inner[:len(inner)-1]
		}
		if len(inner) > 0 && inner[len(inner)-1] == '?' {
			continue
		}

		spaceIdx := bytes.IndexByte(inner, ' ')
		var tagName []byte
		var tagAttrs []byte
		if spaceIdx == -1 {
			tagName = inner
		} else {
			tagName = inner[:spaceIdx]
			tagAttrs = inner[spaceIdx+1:]
		}

		if !isEnd {
			// Start Element Processing
			if bytes.Equal(tagName, tComponent) {
				isCompStack = append(isCompStack, true)
				var cid, class, macro, owner, name, code, state, readStatus string

				parseAttrs(tagAttrs, func(k, v []byte) {
					switch {
					case bytes.Equal(k, bID):
						cid = string(v)
					case bytes.Equal(k, bClass):
						class = string(v)
					case bytes.Equal(k, bMacro):
						macro = string(v)
					case bytes.Equal(k, bOwner):
						owner = string(v)
					case bytes.Equal(k, bName):
						name = string(v)
					case bytes.Equal(k, bCode):
						code = string(v)
					case bytes.Equal(k, bState):
						state = string(v)
					case bytes.Equal(k, bRead):
						readStatus = string(v)
					}
				})

				parentID := ""
				if len(parentStack) > 0 {
					parentID = parentStack[len(parentStack)-1]
				}

				isShip := isShipClass(class)
				isStation := isStationClass(class)
				isVault := isVaultClass(class)
				isModule := isModuleClass(class)

				hasPos := pendingPos.X != 0 || pendingPos.Y != 0 || pendingPos.Z != 0
				info := componentInfo{
					parent: parentID,
					class:  class,
					macro:  macro,
					code:   code,
					pos:    pendingPos,
					hasPos: hasPos,
				}
				pendingPos = Vector3{} // Reset for next component

				if cid != "" {
					idToInfo[cid] = info
				}
				parentStack = append(parentStack, cid)

				if findKhaak && owner == "khaak" && (isStation || isModule) {
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
							ID: cid, Name: s.getName(macro, ""), Type: stype, Macro: macro, IsWreck: state == "wreck",
						})
					}
				}

				if findShips && isShip {
					results.Ships = append(results.Ships, ScanResult{
						ID: cid, Name: s.getName(macro, name), Class: class, Macro: macro,
						Owner: owner, Code: code, IsWreck: state == "wreck",
					})
				}

				if findUnowned && isShip && owner == "ownerless" && state != "wreck" {
					results.Unowned = append(results.Unowned, ScanResult{
						ID: cid, Name: s.getName(macro, name), Class: class,
						Macro: macro, Owner: owner, Code: code,
					})
				}

				if findVault && isVault {
					results.Vaults = append(results.Vaults, ScanResult{
						ID: cid, Name: s.getName(macro, "Data Vault"), Macro: macro,
						IsDecrypted: readStatus == "1",
					})
				}

				if isSelfClosing {
					// Treat it as an immediate end element
					if len(parentStack) > 0 {
						parentStack = parentStack[:len(parentStack)-1]
					}
					if len(isCompStack) > 0 {
						isCompStack = isCompStack[:len(isCompStack)-1]
					}
				}

			} else if bytes.Equal(tagName, tConnection) {
				isCompStack = append(isCompStack, false)
				if isSelfClosing {
					if len(isCompStack) > 0 {
						isCompStack = isCompStack[:len(isCompStack)-1]
					}
					pendingPos = Vector3{}
				}
			} else if bytes.Equal(tagName, tPosition) {
				var pos Vector3
				parseAttrs(tagAttrs, func(k, v []byte) {
					val, _ := strconv.ParseFloat(string(v), 64)
					switch {
					case bytes.Equal(k, bX):
						pos.X = val
					case bytes.Equal(k, bY):
						pos.Y = val
					case bytes.Equal(k, bZ):
						pos.Z = val
					}
				})

				isCompPos := len(isCompStack) > 0 && isCompStack[len(isCompStack)-1]

				if isCompPos && len(parentStack) > 0 {
					cid := parentStack[len(parentStack)-1]
					if info, ok := idToInfo[cid]; ok {
						info.pos.X += pos.X
						info.pos.Y += pos.Y
						info.pos.Z += pos.Z
						info.hasPos = true
						idToInfo[cid] = info
					}
				} else {
					pendingPos.X += pos.X
					pendingPos.Y += pos.Y
					pendingPos.Z += pos.Z
				}
			} else if bytes.Equal(tagName, tGame) || bytes.Equal(tagName, tPlayer) || bytes.Equal(tagName, tSave) {
				keyPrefix := string(tagName) + "_"
				parseAttrs(tagAttrs, func(k, v []byte) {
					results.Info[keyPrefix+string(k)] = string(v)
				})
			}
		} else {
			// End Element Processing
			if bytes.Equal(tagName, tComponent) {
				if len(parentStack) > 0 {
					parentStack = parentStack[:len(parentStack)-1]
				}
				if len(isCompStack) > 0 {
					isCompStack = isCompStack[:len(isCompStack)-1]
				}
			} else if bytes.Equal(tagName, tConnection) {
				if len(isCompStack) > 0 {
					isCompStack = isCompStack[:len(isCompStack)-1]
				}
				pendingPos = Vector3{} // Clear any unconsumed offset to prevent leakage
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
