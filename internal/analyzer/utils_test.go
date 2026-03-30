// Package analyzer provides tools for analyzing X4: Foundations save games.
package analyzer

import "testing"

func TestFormatCredits(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"10000", "100.00 Cr"},
		{"0", "0.00 Cr"},
		{"", "N/A"},
		{"invalid", "N/A"},
	}

	for _, tt := range tests {
		if got := FormatCredits(tt.input); got != tt.expected {
			t.Errorf("FormatCredits(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatGameTime(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0", "0s"},
		{"60", "1m 0s"},
		{"3661", "1h 1m 1s"},
		{"90061", "1d 1h 1m 1s"},
		{"", "N/A"},
		{"invalid", "N/A"},
	}

	for _, tt := range tests {
		if got := FormatGameTime(tt.input); got != tt.expected {
			t.Errorf("FormatGameTime(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatCoord(t *testing.T) {
	if got := FormatCoord(1000.0); got != "1.0km" {
		t.Errorf("FormatCoord(1000.0) = %q; want \"1.0km\"", got)
	}
}
