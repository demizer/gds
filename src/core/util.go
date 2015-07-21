package core

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// sha1sum gets the sha1 hash of filePath using an external hashing tool.
func sha1sum(filePath string) (string, error) {
	cmd := exec.Command("/usr/bin/sha1sum", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("sha1sum error - %s", err.Error())
	}
	return strings.Fields(out.String())[0], err
}

func stripDotDot(path string) string {
	return strings.Replace(filepath.Clean(path), "../", "", -1)
}
