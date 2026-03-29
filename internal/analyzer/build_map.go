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

type tPage struct {
	ID string `xml:"id,attr"`
	T  []struct {
		ID    string `xml:"id,attr"`
		Value string `xml:",chardata"`
	} `xml:"t"`
}

type tFile struct {
	Pages []tPage `xml:"page"`
}

func BuildMacroMap(gameDataDir string) (map[string]string, error) {
	fmt.Printf("Building macro map from: %s\n", gameDataDir)

	allTrans := make(map[string]map[string]string)
	mappings := make(map[string][2]string)

	reTag := regexp.MustCompile(`\{(\d+),(\d+)\}`)

	// 1. Load translations
	filepath.Walk(gameDataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		base := strings.ToLower(filepath.Base(path))
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
		}
		return nil
	})

	// 2. Scan for macro mappings
	filepath.Walk(gameDataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".xml") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

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
						if len(match) == 3 {
							// Find nearest ancestor with a name
							for i := len(nameStack) - 2; i >= 0; i-- {
								if nameStack[i] != "" {
									mLower := strings.ToLower(nameStack[i])
									if strings.Contains(mLower, "ship") || strings.Contains(mLower, "cluster") || strings.Contains(mLower, "sector") || strings.Contains(mLower, "station") || strings.Contains(mLower, "vault") {
										page := strings.TrimLeft(match[1], "0")
										if page == "" && match[1] != "" {
											page = "0"
										}
										mappings[mLower] = [2]string{page, match[2]}
									}
									break
								}
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
		return nil
	})

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

	// 3. Resolve names
	finalMap := make(map[string]string)
	for macro, ids := range mappings {
		if page, ok := allTrans[ids[0]]; ok {
			if text, ok := page[ids[1]]; ok {
				resolved := resolveText(text, allTrans, 0)
				if resolved != "" {
					finalMap[macro] = resolved
				}
			}
		}
	}

	return finalMap, nil
}

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

	if strings.Contains(res, "(") && strings.Contains(res, ")") {
		if strings.Contains(text, "{") {
			reParen := regexp.MustCompile(`\(([^)]+)\)`)
			m := reParen.FindStringSubmatch(res)
			if len(m) == 2 {
				return strings.TrimSpace(m[1])
			}
		}
	}

	res = reTag.ReplaceAllString(res, "")
	reParenCleanup := regexp.MustCompile(`\([^)]+\)`)
	res = reParenCleanup.ReplaceAllString(res, "")

	return strings.TrimSpace(res)
}

func SaveMacroMap(filePath string, macroMap map[string]string) error {
	data, err := json.MarshalIndent(macroMap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}
