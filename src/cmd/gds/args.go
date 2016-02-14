package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

var defaultConfig = `# Generic Device Storage Configuration File
# Use: \df -B1 <mountpoint> to find correct available space in bytes.
# Undersize the device by 1MiB (more or less), otherwise errors will occurr.
backupPath: "/mnt/data"
# Set the number of concurrent device backups. 1 == one device, 2 == two devices
outputStreams: 1
# Device size amounts must be in bytes
# devices:
#   - name: "Test Drive 1"
#     sizeTotal: 4965185763
#     mountPoint: "/mnt/backup1"
#   - name: "Test Drive 2"
#     sizeTotal: 4965185763
#     mountPoint: "/mnt/backup2"
`

// getConfigFile ensures a config file, empty or not, is ready to use.
func getConfigFile(path string) (p string, err error) {
	createConf := func(p string) error {
		err := ioutil.WriteFile(p, []byte(defaultConfig), 0644)
		if err != nil {
			return err
		}
		return nil
	}
	confPath := cleanPath(path)
	ext := filepath.Ext(confPath)
	if ext == "" {
		// confPath does not have an extenision, so maybe it is only a path
		confPath = filepath.Join(confPath, GDS_CONFIG_NAME)
	}
	if _, err = os.Lstat(confPath); err != nil {
		dir := filepath.Dir(confPath)
		if _, err = os.Lstat(dir); err != nil {
			err = os.MkdirAll(dir, 0755)
		}
		if err == nil {
			err = createConf(confPath)
		}
	}
	if err != nil {
		err = fmt.Errorf("Error getting %q: %s", confPath, err.Error())
		confPath = ""
	}
	return filepath.Clean(confPath), err
}

// getContextFile ensures a context file path exists.
func getContextFile(path string) (p string, err error) {
	p = cleanPath(path)
	ext := filepath.Ext(p)
	if ext == "" {
		p = filepath.Join(p, GDS_CONTEXT_FILENAME)
	}
	if _, err = os.Lstat(p); err != nil {
		dir := filepath.Dir(p)
		if _, err = os.Lstat(dir); err != nil {
			err = os.MkdirAll(dir, 0755)
		}
		var f *os.File
		f, err = os.Create(p)
		f.Close()
	}
	if err != nil {
		err = fmt.Errorf("Could not get context file %q: %s", p, err.Error())
	}
	return
}
