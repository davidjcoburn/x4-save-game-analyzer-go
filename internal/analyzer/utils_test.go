// Package analyzer provides tools for analyzing X4: Foundations save games.
package analyzer

import "testing"

func TestFormatCredits(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"10000", "100.00 Cr"},
		{"1000000", "10,000.00 Cr"},
		{"506864771", "5,068,647.71 Cr"},
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
		{"3605", "1h 0m 5s"},
		{"3661", "1h 1m 1s"},
		{"90061", "1d 1h 1m 1s"},
		{"", "N/A"},
		{"invalid", "N/A"},
		{"-100", "N/A"},
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
	if got := FormatCoord(-0.0001); got != "0.0km" {
		t.Errorf("FormatCoord(-0.0001) = %q; want \"0.0km\"", got)
	}
	if got := FormatCoord(-1500.0); got != "-1.5km" {
		t.Errorf("FormatCoord(-1500.0) = %q; want \"-1.5km\"", got)
	}
}
