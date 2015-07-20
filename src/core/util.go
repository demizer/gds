package core

import (
	"bytes"
	"os/exec"
	"strings"
)

// sha1sum gets the sha1 hash of filePath using an external hashing tool.
func sha1sum(filePath string) (string, error) {
	cmd := exec.Command("/usr/bin/sha1sum", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.Fields(out.String())[0], err
}
