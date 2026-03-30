// Package analyzer provides tools for analyzing X4: Foundations save games.
package analyzer

import (
	"strings"
	"testing"
)

// TestBasicScan verifies the core scanning and hierarchy resolution logic using a mock XML stream.
func TestBasicScan(t *testing.T) {
	sampleXML := `<?xml version="1.0" encoding="UTF-8"?>
<save>
    <info>
        <game version="600" build="500000" name="X4"/>
        <player name="TestPlayer" money="100000"/>
        <save date="1672531200"/>
    </info>
    <universe>
        <component id="gal" class="galaxy">
            <component id="clu" class="cluster" macro="cluster_name">
                <component id="sec" class="sector" macro="sector_name" code="SEC_01">
                    <offset><position x="1000" y="2000" z="3000"/></offset>
                    <component id="ship1" class="ship_l" macro="ship_macro" owner="player" name="MyShip">
                        <offset><position x="100" y="200" z="300"/></offset>
                    </component>
                </component>
            </component>
        </component>
    </universe>
</save>
`

	scanner := NewX4SaveScanner("dummy.xml")
	// Mock macroMap
	scanner.macroMap = map[string]string{"cluster_name": "Test System"}

	reader := strings.NewReader(sampleXML)
	results, err := scanner.ScanReader(reader, "MyShip", false, false, false)
	if err != nil {
		t.Fatalf("ScanReader failed: %v", err)
	}

	// Verify info
	if results.Info["player_name"] != "TestPlayer" {
		t.Errorf("Expected player_name TestPlayer, got %s", results.Info["player_name"])
	}
	if results.Info["player_money"] != "100000" {
		t.Errorf("Expected player_money 100000, got %s", results.Info["player_money"])
	}

	// Verify ship
	if len(results.Ships) != 1 {
		t.Fatalf("Expected 1 ship, got %d", len(results.Ships))
	}
	ship := results.Ships[0]
	if ship.Name != "MyShip" {
		t.Errorf("Expected ship name MyShip, got %s", ship.Name)
	}
	if ship.Owner != "player" {
		t.Errorf("Expected owner player, got %s", ship.Owner)
	}
	if ship.Sector != "SEC_01" {
		t.Errorf("Expected sector SEC_01, got %s", ship.Sector)
	}
	if ship.System != "Test System" {
		t.Errorf("Expected system Test System, got %s", ship.System)
	}

	// Verify global position (1000+100, 2000+200, 3000+300)
	if ship.Pos.X != 1100 || ship.Pos.Y != 2200 || ship.Pos.Z != 3300 {
		t.Errorf("Expected global pos (1100, 2200, 3300), got (%v, %v, %v)", ship.Pos.X, ship.Pos.Y, ship.Pos.Z)
	}
}

// TestExtendedScan verifies scanning for Kha'ak, unowned ships, and data vaults.
func TestExtendedScan(t *testing.T) {
	sampleXML := `<?xml version="1.0" encoding="UTF-8"?>
<save>
    <universe>
        <component id="clu" class="cluster" macro="cluster_name">
            <component id="sec" class="sector" macro="sector_name" code="SEC_01">
                <component id="k1" class="station" macro="station_khk_hive_macro" owner="khaak"/>
                <component id="u1" class="ship_m" macro="abandoned_ship_macro" owner="ownerless"/>
                <component id="v1" class="datavault" macro="vault_macro" read="1"/>
            </component>
        </component>
    </universe>
</save>
`

	scanner := NewX4SaveScanner("dummy.xml")
	// Mock macroMap
	scanner.macroMap = map[string]string{
		"cluster_name": "Test System",
		"vault_macro":  "Data Vault",
	}

	reader := strings.NewReader(sampleXML)
	results, err := scanner.ScanReader(reader, "", true, true, true)
	if err != nil {
		t.Fatalf("ScanReader failed: %v", err)
	}

	// Verify Kha'ak
	if len(results.Khaak) != 1 {
		t.Errorf("Expected 1 Kha'ak target, got %d", len(results.Khaak))
	} else if results.Khaak[0].Type != "Hive" {
		t.Errorf("Expected Kha'ak type Hive, got %s", results.Khaak[0].Type)
	}

	// Verify Unowned
	if len(results.Unowned) != 1 {
		t.Errorf("Expected 1 unowned ship, got %d", len(results.Unowned))
	}

	// Verify Vaults
	if len(results.Vaults) != 1 {
		t.Errorf("Expected 1 data vault, got %d", len(results.Vaults))
	} else if !results.Vaults[0].IsDecrypted {
		t.Errorf("Expected vault to be decrypted")
	}
}
