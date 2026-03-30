// Package analyzer provides tools for analyzing X4: Foundations save games.
package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildMacroMap(t *testing.T) {
	// Create a temporary directory for test data
	tmpDir, err := os.MkdirTemp("", "game-data-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock translation file
	transDir := tmpDir
	transFile := filepath.Join(transDir, "0001.xml")
	transContent := `<?xml version="1.0" encoding="UTF-8"?>
<language id="44">
    <page id="20001" name="macros">
        <t id="101">Test Ship Name</t>
    </page>
</language>
`
	if err := os.WriteFile(transFile, []byte(transContent), 0644); err != nil {
		t.Fatalf("Failed to write mock translation file: %v", err)
	}

	// Create a mock macro file
	macroFile := filepath.Join(tmpDir, "test_macro.xml")
	macroContent := `<?xml version="1.0" encoding="UTF-8"?>
<macros>
    <macro name="test_ship_macro" class="ship_l">
        <identification name="{20001,101}"/>
    </macro>
</macros>
`
	if err := os.WriteFile(macroFile, []byte(macroContent), 0644); err != nil {
		t.Fatalf("Failed to write mock macro file: %v", err)
	}

	// Run BuildMacroMap
	macroMap, err := BuildMacroMap(tmpDir, nil)
	if err != nil {
		t.Fatalf("BuildMacroMap failed: %v", err)
	}

	// Verify the result
	if got := macroMap["test_ship_macro"]; got != "Test Ship Name" {
		t.Errorf("Expected macro mapping 'Test Ship Name', got %q", got)
	}

	// Verify manual override
	if got := macroMap["landmarks_kha_nest_01_macro"]; got != "Kha'ak Installation" {
		t.Errorf("Expected manual override 'Kha'ak Installation', got %q", got)
	}
}
