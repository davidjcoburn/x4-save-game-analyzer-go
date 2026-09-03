// Package analyzer provides tools for analyzing X4: Foundations save games.
package analyzer

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// formatWithCommas formats a float64 with 2 decimal places and thousands separators.
func formatWithCommas(val float64) string {
	sign := ""
	if val < 0 {
		sign = "-"
		val = -val
	}
	intPart := int64(val)
	fracPart := int64(math.Round((val - float64(intPart)) * 100))
	if fracPart >= 100 {
		intPart++
		fracPart = 0
	}
	intStr := strconv.FormatInt(intPart, 10)
	var sb strings.Builder
	sb.WriteString(sign)
	n := len(intStr)
	for i, c := range intStr {
		if i > 0 && (n-i)%3 == 0 {
			sb.WriteByte(',')
		}
		sb.WriteRune(c)
	}
	sb.WriteByte('.')
	if fracPart < 10 {
		sb.WriteByte('0')
	}
	sb.WriteString(strconv.FormatInt(fracPart, 10))
	return sb.String()
}

// FormatCredits converts a centicredit string value to a human-readable credit format.
func FormatCredits(centiCredits string) string {
	if centiCredits == "" {
		return "N/A"
	}
	val, err := strconv.ParseFloat(centiCredits, 64)
	if err != nil || math.IsNaN(val) || math.IsInf(val, 0) {
		return "N/A"
	}
	creditsVal := val / CentiCreditToCredit
	return formatWithCommas(creditsVal) + " Cr"
}

// FormatGameTime converts a duration in seconds (as string) to a human-readable day/hour/minute/second format.
func FormatGameTime(secondsStr string) string {
	if secondsStr == "" {
		return "N/A"
	}
	seconds, err := strconv.ParseFloat(secondsStr, 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 {
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
	if minutes > 0 || ((days > 0 || hours > 0) && secs > 0) {
		res += fmt.Sprintf("%dm ", minutes)
	}
	res += fmt.Sprintf("%ds", secs)

	return res
}

// FormatCoord converts a coordinate value from meters to a human-readable kilometer format.
func FormatCoord(v float64) string {
	km := v / MeterToKm
	if math.Abs(km) < 0.05 {
		km = 0.0
	}
	return fmt.Sprintf("%.1fkm", km)
}
