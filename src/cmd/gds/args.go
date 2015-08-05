package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

var defaultConfig = `# Ghetto Device Storage Configuration File
# Use: \df -B1 <mountpoint> to find correct available space in bytes.
# Undersize the device by 1MiB (more or less), otherwise errors will occurr.
backupPath: "/mnt/data"
# Set the number of concurrent device backups. 1 == one device, 2 == two devices
outputStreams: 1
# Device size amounts must be in bytes
# devices:
#   - name: "Test Drive 1"
#     size: 4965185763
#     mountPoint: "/mnt/backup1"
#   - name: "Test Drive 2"
#     size: 4965185763
#     mountPoint: "/mnt/backup2"
`

// getConfigFile ensures a config file, empty or not, is ready to use.
func getConfigFile(path string) (string, error) {
	var err error

	createConf := func(p string) error {
		err := ioutil.WriteFile(p, []byte(defaultConfig), 0644)
		if err != nil {
			return err
		}
		return nil
	}

	confPath := cleanPath(path)
	ext := filepath.Ext(path)
	log.WithFields(logrus.Fields{
		"extension": ext,
	}).Debug("Config extension")
	if ext != ".yml" && ext != ".yaml" {
		confPath = filepath.Join(confPath, "config.yml")
	}

	if _, err = os.Lstat(confPath); err != nil {
		dir := filepath.Dir(confPath)
		if _, err = os.Lstat(dir); err == nil {
			err = createConf(confPath)
		} else {
			err = os.MkdirAll(dir, 0755)
			if err == nil {
				err = createConf(confPath)
			}
		}
	}

	if err != nil {
		err = fmt.Errorf("Error getting %q: %s", confPath, err.Error())
		confPath = ""
	}

	return filepath.Clean(confPath), err
}
