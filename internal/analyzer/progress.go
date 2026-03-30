// Package analyzer provides tools for analyzing X4: Foundations save games.
package analyzer

import (
	"io"
)

// ProgressReader wraps an io.Reader and reports progress via a callback.
type ProgressReader struct {
	io.Reader
	total     int64
	current   int64
	lastSent  float64
	onPercent func(float64)
}

// NewProgressReader creates a new ProgressReader.
func NewProgressReader(r io.Reader, total int64, onPercent func(float64)) *ProgressReader {
	return &ProgressReader{
		Reader:    r,
		total:     total,
		onPercent: onPercent,
	}
}

// Read implements the io.Reader interface and tracks the number of bytes read to report progress.
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	if n > 0 {
		pr.current += int64(n)
		if pr.total > 0 && pr.onPercent != nil {
			percent := float64(pr.current) / float64(pr.total) * 100
			if percent-pr.lastSent >= 1.0 { // Update every 1%
				pr.onPercent(percent)
				pr.lastSent = percent
			}
		}
	}
	return n, err
}
