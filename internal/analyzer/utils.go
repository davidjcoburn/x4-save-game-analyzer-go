package analyzer

import (
	"fmt"
	"math"
	"strconv"
)

func FormatCredits(centiCredits string) string {
	if centiCredits == "" {
		return "N/A"
	}
	val, err := strconv.ParseFloat(centiCredits, 64)
	if err != nil {
		return "N/A"
	}
	creditsVal := val / CentiCreditToCredit
	return fmt.Sprintf("%.2f Cr", creditsVal)
}

func FormatGameTime(secondsStr string) string {
	if secondsStr == "" {
		return "N/A"
	}
	seconds, err := strconv.ParseFloat(secondsStr, 64)
	if err != nil {
		return "N/A"
	}

	days := int(math.Floor(seconds / SecondsPerDay))
	remaining := math.Mod(seconds, SecondsPerDay)
	hours := int(math.Floor(remaining / SecondsPerHour))
	remaining = math.Mod(remaining, SecondsPerHour)
	minutes := int(math.Floor(remaining / SecondsPerMinute))
	secs := int(math.Mod(remaining, SecondsPerMinute))

	res := ""
	if days > 0 {
		res += fmt.Sprintf("%dd ", days)
	}
	if hours > 0 {
		res += fmt.Sprintf("%dh ", hours)
	}
	if minutes > 0 {
		res += fmt.Sprintf("%dm ", minutes)
	}
	res += fmt.Sprintf("%ds", secs)

	return res
}

func FormatCoord(v float64) string {
	return fmt.Sprintf("%.1fkm", v/MeterToKm)
}
