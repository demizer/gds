package main

import (
	"os"
	"path/filepath"
	"strings"
)

// cleanPath returns a path string that is clean. ~, ~/, and $HOME are replaced with the proper expansions
func cleanPath(path string) string {
	nPath := filepath.Clean(path)
	if strings.Contains(nPath, "$HOME") {
		nPath = strings.Replace(nPath, "$HOME", os.Getenv("HOME"), -1)
	}
	if strings.Contains(nPath, "~/") {
		nPath = strings.Replace(nPath, "~", os.Getenv("HOME"), -1)
	}
	if strings.Contains(nPath, "~") {
		nPath = strings.Replace(nPath, "~", "/home/", -1)
	}
	return nPath
}
