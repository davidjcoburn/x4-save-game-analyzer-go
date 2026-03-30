// Package analyzer provides tools for analyzing X4: Foundations save games.
package analyzer

// Constants for scaling X4 save data to more human-readable formats.
const (
	// MeterToKm is the conversion factor from meters to kilometers.
	MeterToKm = 1000.0
	// CentiCreditToCredit is the conversion factor from centicredits to credits.
	CentiCreditToCredit = 100.0
	// SecondsPerDay is the number of seconds in a day.
	SecondsPerDay = 86400
	// SecondsPerHour is the number of seconds in an hour.
	SecondsPerHour = 3600
	// SecondsPerMinute is the number of seconds in a minute.
	SecondsPerMinute = 60
)

// Interesting component classes for hierarchy tracking.
var (
	shipClasses    = map[string]bool{"ship": true, "ship_s": true, "ship_m": true, "ship_l": true, "ship_xl": true}
	stationClasses = map[string]bool{"station": true, "weaponplatform": true}
	vaultClasses   = map[string]bool{"datavault": true}
)

// Kha'ak station types substr patterns.
var (
	khaakHiveSubstr         = []string{"hive"}
	khaakNestSubstr         = []string{"nest"}
	khaakInstallationSubstr = []string{"installation", "raid", "station_khk", "landmarks_kha"}
)
