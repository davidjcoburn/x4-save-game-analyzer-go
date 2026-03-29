//go:build windows

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func openFileDialog() (string, error) {
	// Try to find the default X4 save directory
	home, _ := os.UserHomeDir()
	initialDir := filepath.Join(home, "Documents", "Egosoft", "X4")
	if _, err := os.Stat(initialDir); os.IsNotExist(err) {
		initialDir, _ = os.Getwd()
	}

	// PowerShell script to show OpenFileDialog
	// We use -Command with a script block that loads Windows.Forms
	psScript := fmt.Sprintf(`
		Add-Type -AssemblyName System.Windows.Forms
		$f = New-Object System.Windows.Forms.OpenFileDialog
		$f.InitialDirectory = '%s'
		$f.Filter = 'X4 Save (*.xml.gz)|*.xml.gz|XML Files (*.xml)|*.xml|All Files (*.*)|*.*'
		$f.Title = 'Select X4 Save Game'
		$res = $f.ShowDialog()
		if($res -eq 'OK') { $f.FileName }
	`, initialDir)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}
