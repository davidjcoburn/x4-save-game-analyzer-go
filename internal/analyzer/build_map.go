// Package analyzer provides tools for analyzing X4: Foundations save games.
package analyzer

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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

// BuildMacroMap scans the game data directory to build a mapping from macro names to human-readable names.
func BuildMacroMap(gameDataDir string, onProgress func(float64)) (map[string]string, error) {
	fmt.Printf("Building macro map from: %s\n", gameDataDir)

	allTrans := make(map[string]map[string]string)
	mappings := make(map[string][2]string)

	reTag := regexp.MustCompile(`\{(\d+),(\d+)\}`)

	// 1. Collect all XML files in a single pass
	type xmlFile struct {
		path string
		base string
	}
	var files []xmlFile
	filepath.WalkDir(gameDataDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".xml") {
			files = append(files, xmlFile{path: path, base: strings.ToLower(filepath.Base(path))})
		}
		return nil
	})

	totalFiles := int64(len(files))
	var lastSent float64

	// 2. Process collected files
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

		// 1. Process translations
		if base == "0001-l044.xml" || base == "0001.xml" {
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
			continue
		}

		// 2. Process wares.xml for ship names
		if base == "wares.xml" {
			f, err := os.Open(path)
			if err != nil {
				continue
			}
			decoder := xml.NewDecoder(f)
			var currentWareName string
			for {
				token, err := decoder.Token()
				if err == io.EOF {
					break
				}
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
							if match := reTag.FindStringSubmatch(currentWareName); len(match) == 3 {
								page := strings.TrimLeft(match[1], "0")
								if page == "" && match[1] != "" {
									page = "0"
								}
								mappings[macroName] = [2]string{page, match[2]}
							} else if !strings.Contains(currentWareName, "{") {
								mappings[macroName] = [2]string{"LITERAL", currentWareName}
							}
						}
					}
				case xml.EndElement:
					if t.Name.Local == "ware" {
						currentWareName = ""
					}
				}
			}
			f.Close()
			continue
		}

		// 3. Scan for macro mappings in other XML files
		f, err := os.Open(path)
		if err != nil {
			continue
		}

		decoder := xml.NewDecoder(f)
		var nameStack []string

		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
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
						match := reTag.FindStringSubmatch(identName)
						// Find nearest ancestor with a name
						var ancestorName string
						for i := len(nameStack) - 2; i >= 0; i-- {
							if nameStack[i] != "" {
								ancestorName = strings.ToLower(nameStack[i])
								break
							}
						}

						if ancestorName != "" && (strings.Contains(ancestorName, "ship") || strings.Contains(ancestorName, "cluster") || strings.Contains(ancestorName, "sector") || strings.Contains(ancestorName, "station") || strings.Contains(ancestorName, "vault") || strings.Contains(ancestorName, "kha") || strings.Contains(ancestorName, "module")) {
							if len(match) == 3 {
								page := strings.TrimLeft(match[1], "0")
								if page == "" && match[1] != "" {
									page = "0"
								}
								mappings[ancestorName] = [2]string{page, match[2]}
							} else if !strings.Contains(identName, "{") {
								// Literal name (common in mods or special ships)
								mappings[ancestorName] = [2]string{"LITERAL", identName}
							}
						}
					}
				}
			case xml.EndElement:
				if len(nameStack) > 0 {
					nameStack = nameStack[:len(nameStack)-1]
				}
			}
		}
		f.Close()
	}

	if onProgress != nil {
		onProgress(100.0)
	}

	// Heuristic for base game clusters if still missing
	for i := 1; i < 100; i++ {
		m := fmt.Sprintf("cluster_%02d_macro", i)
		if _, ok := mappings[m]; !ok {
			mappings[m] = [2]string{"20003", fmt.Sprintf("%d0001", i)}
		}
		sm := fmt.Sprintf("cluster_%02d_sector001_macro", i)
		if _, ok := mappings[sm]; !ok {
			mappings[sm] = [2]string{"20004", fmt.Sprintf("%d00011", i)}
		}
	}

	// 4. Resolve names
	finalMap := make(map[string]string)
	for macro, ids := range mappings {
		if ids[0] == "LITERAL" {
			finalMap[macro] = ids[1]
			continue
		}
		if page, ok := allTrans[ids[0]]; ok {
			if text, ok := page[ids[1]]; ok {
				resolved := resolveText(text, allTrans, 0)
				if resolved != "" {
					finalMap[macro] = resolved
				}
			}
		}
	}

	// 5. Manual overrides/fallbacks for known macros with poor or missing names
	overrides := map[string]string{
		"landmarks_kha_nest_01_macro":             "Kha'ak Nest Installation",
		"landmarks_kha_hive_01_macro":             "Kha'ak Hive Installation",
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

	reTag := regexp.MustCompile(`\{(\d+),(\d+)\}`)
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
