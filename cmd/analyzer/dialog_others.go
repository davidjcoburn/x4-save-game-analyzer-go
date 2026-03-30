//go:build !windows

// Package main provides the CLI entry point and platform-specific dialogs.
package main

import "errors"

// ErrNotImplemented is returned when a platform-specific feature is not implemented.
var ErrNotImplemented = errors.New("not implemented")

// openFileDialog is a placeholder for non-Windows platforms where file dialogs are not yet implemented.
func openFileDialog() (string, error) {
	return "", ErrNotImplemented
}

// openFolderDialog is a placeholder for non-Windows platforms where folder dialogs are not yet implemented.
func openFolderDialog() (string, error) {
	return "", ErrNotImplemented
}
