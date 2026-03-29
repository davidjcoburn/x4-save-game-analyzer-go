//go:build !windows

package main

import "errors"

func openFileDialog() (string, error) {
	return "", errors.New("open file dialog is only supported on Windows in this version")
}
