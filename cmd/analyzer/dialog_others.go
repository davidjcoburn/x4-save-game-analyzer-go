//go:build !windows

// Package main provides the CLI entry point and platform-specific dialogs.
package main

import "errors"

// openFileDialog is a placeholder for non-Windows platforms where file dialogs are not yet implemented.
func openFileDialog() (string, error) {
	return "", errors.New("open file dialog is only supported on Windows in this version")
}

// openFolderDialog is a placeholder for non-Windows platforms where folder dialogs are not yet implemented.
func openFolderDialog() (string, error) {
	return "", errors.New("open folder dialog is only supported on Windows in this version")
}
