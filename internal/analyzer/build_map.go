// Package analyzer provides tools for analyzing X4: Foundations save games.
package analyzer

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// tPage represents a page of translations in an X4 translation file.
type tPage struct {
	ID string `xml:"id,attr"`
	T  []struct {
		ID    string `xml:"id,attr"`
		Value string `xml:",chardata"`
	} `xml:"t"`
}

// tFile represents an X4 translation file.
type tFile struct {
	Pages []tPage `xml:"page"`
}

var reTag = regexp.MustCompile(`\{(\d+),(\d+)\}`)

// BuildMacroMap scans the game data directory to build a mapping from macro names to human-readable names.
func BuildMacroMap(gameDataDir string, onProgress func(float64)) (map[string]string, error) {
	fmt.Printf("Building macro map from: %s\n", gameDataDir)

	allTrans := make(map[string]map[string]string)
	mappings := make(map[string]string)

	// 1. Collect all XML files in a single pass
	type xmlFile struct {
		path string
		base string
	}
	var files []xmlFile
	var transFiles []string

	filepath.WalkDir(gameDataDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".xml") {
			base := strings.ToLower(filepath.Base(path))
			if base == "0001-l044.xml" || base == "0001.xml" {
				transFiles = append(transFiles, path)
			} else {
				files = append(files, xmlFile{path: path, base: base})
			}
		}
		return nil
	})

	// Sort translation files so base game "main" files load first, followed by extensions/DLCs
	// This ensures DLC and mod translations properly override base game entries.
	sort.Slice(transFiles, func(i, j int) bool {
		iIsMain := strings.Contains(strings.ToLower(transFiles[i]), "main")
		jIsMain := strings.Contains(strings.ToLower(transFiles[j]), "main")
		if iIsMain != jIsMain {
			return iIsMain // main comes first
		}
		return transFiles[i] < transFiles[j]
	})

	for _, path := range transFiles {
		data, err := os.ReadFile(path)
		if err == nil {
			var tf tFile
			if err := xml.Unmarshal(data, &tf); err == nil {
				for _, p := range tf.Pages {
					pid := strings.TrimLeft(p.ID, "0")
					if pid == "" && p.ID != "" {
						pid = "0"
					}
					if _, ok := allTrans[pid]; !ok {
						allTrans[pid] = make(map[string]string)
					}
					for _, t := range p.T {
						allTrans[pid][t.ID] = t.Value
					}
				}
			}
		}
	}

	totalFiles := int64(len(files))
	var lastSent float64

	// 2. Process collected macro and ware files
	for i, fInfo := range files {
		if onProgress != nil && totalFiles > 0 {
			percent := float64(i+1) / float64(totalFiles) * 100
			if percent-lastSent >= 1.0 || i == len(files)-1 {
				onProgress(percent)
				lastSent = percent
			}
		}

		path := fInfo.path
		base := fInfo.base

		// Process wares.xml for ship names
		if base == "wares.xml" {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			decoder := xml.NewDecoder(bytes.NewReader(data))
			var currentWareName string
			for {
				token, err := decoder.Token()
				if err != nil {
					break
				}
				switch t := token.(type) {
				case xml.StartElement:
					if t.Name.Local == "ware" {
						for _, attr := range t.Attr {
							if attr.Name.Local == "name" {
								currentWareName = attr.Value
							}
						}
					} else if t.Name.Local == "component" && currentWareName != "" {
						var macroName string
						for _, attr := range t.Attr {
							if attr.Name.Local == "ref" {
								macroName = strings.ToLower(attr.Value)
							}
						}
						if macroName != "" {
							mappings[macroName] = currentWareName
						}
					}
				case xml.EndElement:
					if t.Name.Local == "ware" {
						currentWareName = ""
					}
				}
			}
			continue
		}

		// Scan for macro mappings in other XML files
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// Fast skip: only parse XML if it contains an identification tag
		if !bytes.Contains(data, []byte("<identification")) {
			continue
		}

		decoder := xml.NewDecoder(bytes.NewReader(data))
		var nameStack []string

		for {
			token, err := decoder.Token()
			if err != nil {
				break
			}

			switch t := token.(type) {
			case xml.StartElement:
				name := ""
				for _, attr := range t.Attr {
					if attr.Name.Local == "name" || attr.Name.Local == "macro" {
						name = attr.Value
					}
				}
				nameStack = append(nameStack, name)

				if t.Name.Local == "identification" {
					var identName string
					for _, attr := range t.Attr {
						if attr.Name.Local == "name" {
							identName = attr.Value
						}
					}
					if identName != "" {
						var ancestorName string
						for j := len(nameStack) - 2; j >= 0; j-- {
							if nameStack[j] != "" {
								ancestorName = strings.ToLower(nameStack[j])
								break
							}
						}

						if ancestorName != "" && (strings.Contains(ancestorName, "ship") || strings.Contains(ancestorName, "cluster") || strings.Contains(ancestorName, "sector") || strings.Contains(ancestorName, "station") || strings.Contains(ancestorName, "vault") || strings.Contains(ancestorName, "kha") || strings.Contains(ancestorName, "module")) {
							mappings[ancestorName] = identName
						}
					}
				}
			case xml.EndElement:
				if len(nameStack) > 0 {
					nameStack = nameStack[:len(nameStack)-1]
				}
			}
		}
	}

	if onProgress != nil {
		onProgress(100.0)
	}

	// Heuristic for base game clusters if still missing
	for i := 1; i < 100; i++ {
		m := fmt.Sprintf("cluster_%02d_macro", i)
		if _, ok := mappings[m]; !ok {
			mappings[m] = fmt.Sprintf("{20003,%d0001}", i)
		}
		sm := fmt.Sprintf("cluster_%02d_sector001_macro", i)
		if _, ok := mappings[sm]; !ok {
			mappings[sm] = fmt.Sprintf("{20004,%d00011}", i)
		}
	}

	// 3. Resolve names
	finalMap := make(map[string]string)
	for macro, rawIdent := range mappings {
		resolved := resolveText(rawIdent, allTrans, 0)
		if resolved != "" {
			finalMap[macro] = resolved
		}
	}

	// 4. Manual overrides/fallbacks for known macros with poor or missing names
	overrides := map[string]string{
		"landmarks_kha_nest_01_macro":           "Kha'ak Nest Installation",
		"landmarks_kha_hive_01_macro":           "Kha'ak Hive Installation",
		"landmarks_kha_weaponplatform_01_macro": "Kha'ak Weapon Platform",
		"ship_par_l_expeditionary_01_euh_macro": "Trinity",
	}
	for m, name := range overrides {
		if _, ok := finalMap[m]; !ok {
			finalMap[m] = name
		}
	}

	return finalMap, nil
}

// resolveText recursively resolves X4 translation tags like {page,id} in strings.
func resolveText(text string, allTrans map[string]map[string]string, depth int) string {
	if depth > 10 {
		return text
	}

	res := reTag.ReplaceAllStringFunc(text, func(m string) string {
		match := reTag.FindStringSubmatch(m)
		if len(match) == 3 {
			page := strings.TrimLeft(match[1], "0")
			if page == "" && match[1] != "" {
				page = "0"
			}
			if p, ok := allTrans[page]; ok {
				if val, ok := p[match[2]]; ok {
					return resolveText(val, allTrans, depth+1)
				}
			}
		}
		return m
	})

	// Unescape literal parentheses and brackets first so we can see the real text
	res = strings.ReplaceAll(res, `\(`, "(")
	res = strings.ReplaceAll(res, `\)`, ")")
	res = strings.ReplaceAll(res, `\{`, "{")
	res = strings.ReplaceAll(res, `\}`, "}")

	// X4 translation files often use (Comment) for voice or translation hints.
	// We want to strip these if they are redundant.
	
	// 1. Strip (Comment) from the very beginning if it repeats what follows.
	if strings.HasPrefix(res, "(") {
		count := 0
		endIdx := -1
		for i, c := range res {
			if c == '(' {
				count++
			} else if c == ')' {
				count--
				if count == 0 {
					endIdx = i
					break
				}
			}
		}
		if endIdx != -1 && endIdx < len(res)-1 {
			inner := res[1:endIdx]
			remaining := strings.TrimSpace(res[endIdx+1:])
			if remaining != "" {
				if strings.Contains(strings.ToLower(remaining), strings.ToLower(inner)) || len(inner) > 20 {
					res = remaining
				}
			}
		}
	}

	// 2. Strip (Comment) from the very end if it repeats what preceded it or is metadata.
	if strings.HasSuffix(res, ")") {
		startIdx := -1
		count := 0
		for i := len(res) - 1; i >= 0; i-- {
			if res[i] == ')' {
				count++
			} else if res[i] == '(' {
				count--
				if count == 0 {
					startIdx = i
					break
				}
			}
		}
		if startIdx != -1 && startIdx > 0 {
			inner := res[startIdx+1 : len(res)-1]
			preceding := strings.TrimSpace(res[:startIdx])
			if preceding != "" {
				lowerInner := strings.ToLower(inner)
				if strings.Contains(strings.ToLower(preceding), lowerInner) ||
					strings.HasPrefix(lowerInner, "voice:") ||
					strings.HasPrefix(lowerInner, "comment:") ||
					strings.HasPrefix(lowerInner, "speak as") ||
					strings.HasPrefix(lowerInner, "do not translate") ||
					len(inner) > 30 {
					res = preceding
				}
			}
		}
	}

	return strings.TrimSpace(res)
}


// SaveMacroMap saves the macro mapping to a JSON file.
func SaveMacroMap(filePath string, macroMap map[string]string) error {
	data, err := json.MarshalIndent(macroMap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}
