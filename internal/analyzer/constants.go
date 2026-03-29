package analyzer

// Scale factors for X4 save data
const (
	MeterToKm             = 1000.0
	CentiCreditToCredit   = 100.0
	SecondsPerDay         = 86400
	SecondsPerHour        = 3600
	SecondsPerMinute      = 60
)

// Interesting component classes for hierarchy tracking
var (
	shipClasses    = map[string]bool{"ship": true, "ship_s": true, "ship_m": true, "ship_l": true, "ship_xl": true}
	stationClasses = map[string]bool{"station": true}
	vaultClasses   = map[string]bool{"datavault": true}
)

// Kha'ak station types
var (
	khaakHiveSubstr         = []string{"hive"}
	khaakNestSubstr         = []string{"nest"}
	khaakInstallationSubstr = []string{"installation", "raid", "station_khk", "landmarks_kha"}
)
