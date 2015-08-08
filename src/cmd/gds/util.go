package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"
)

func checkEnvVariables(c *cli.Context) (err error) {
	cd := os.Getenv("GDS_CONFIG_DIR")
	if cd == "" {
		GDS_CONFIG_DIR = cleanPath(c.GlobalString("config-dir"))
		err = os.Setenv("GDS_CONFIG_DIR", GDS_CONFIG_DIR)
	} else {
		GDS_CONFIG_DIR = cd
	}
	return err
}

// cleanPath returns a path string that is clean. ~, ~/, and $HOME are replaced with the proper expansions
func cleanPath(path string) string {
	nPath := filepath.Clean(path)
	if strings.Contains(nPath, "$HOME") {
		nPath = strings.Replace(nPath, "$HOME", os.Getenv("HOME"), -1)
	}
	if strings.Contains(nPath, "$GDS_CONFIG_DIR") {
		nPath = strings.Replace(nPath, "$GDS_CONFIG_DIR", os.Getenv("GDS_CONFIG_DIR"), -1)
	}
	if strings.Contains(nPath, "~/") {
		nPath = strings.Replace(nPath, "~", os.Getenv("HOME"), -1)
	}
	if strings.Contains(nPath, "~") {
		nPath = strings.Replace(nPath, "~", "/home/", -1)
	}
	return nPath
}
