//go:build windows

// Package main provides the CLI entry point and platform-specific dialogs.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// openFileDialog opens a Windows file selection dialog using PowerShell to allow the user to pick an X4 save file.
func openFileDialog() (string, error) {
	// Try to find the default X4 save directory
	home, _ := os.UserHomeDir()
	initialDir := filepath.Join(home, "Documents", "Egosoft", "X4")
	if _, err := os.Stat(initialDir); os.IsNotExist(err) {
		initialDir, _ = os.Getwd()
	}

	// PowerShell script to show OpenFileDialog.
	// We use an environment variable to pass the initial directory safely.
	psScript := `
		$ErrorActionPreference = 'Stop'
		Add-Type -AssemblyName System.Windows.Forms
		$f = New-Object System.Windows.Forms.OpenFileDialog
		$f.InitialDirectory = $env:TARGET_INIT_DIR
		$f.Filter = 'X4 Save (*.xml.gz)|*.xml.gz|XML Files (*.xml)|*.xml|All Files (*.*)|*.*'
		$f.Title = 'Select X4 Save Game'
		$res = $f.ShowDialog()
		if($res -eq 'OK') { $f.FileName }
	`

	cmd := exec.Command("powershell", "-NoProfile", "-Sta", "-Command", psScript)
	cmd.Env = append(os.Environ(), "TARGET_INIT_DIR="+initialDir)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		if errOut.Len() > 0 {
			return "", fmt.Errorf("powershell error: %s", strings.TrimSpace(errOut.String()))
		}
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}

// openFolderDialog opens a Windows folder selection dialog using PowerShell to allow the user to select the X4 game root directory.
func openFolderDialog() (string, error) {
	// PowerShell script to show FolderBrowserDialog.
	psScript := `
		Add-Type -AssemblyName System.Windows.Forms
		$f = New-Object System.Windows.Forms.FolderBrowserDialog
		$f.Description = 'Select X4 Foundations Root Directory'
		$f.ShowNewFolderButton = $false
		$res = $f.ShowDialog()
		if($res -eq 'OK') { $f.SelectedPath }
	`

	cmd := exec.Command("powershell", "-NoProfile", "-Sta", "-Command", psScript)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		if errOut.Len() > 0 {
			return "", fmt.Errorf("powershell error: %s", strings.TrimSpace(errOut.String()))
		}
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}
