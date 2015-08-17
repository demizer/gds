package main

import (
	"core"
	"io/ioutil"
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
		nPath = strings.Replace(nPath, "~", "home/", -1)
	}
	return nPath
}

func deviceIsMountedByUUID(mountPoint, uuid string) (bool, error) {
	f, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return false, err
	}
	// key = mountpoint, val = deviceFile
	devs := make(map[string]string)
	for _, v := range strings.Split(string(f), "\n") {
		if strings.Contains(v, mountPoint) {
			devFile := strings.Split(v, " ")[0]
			mnt := strings.Split(v, " ")[1]
			devs[mnt] = devFile
		}
	}
	var found bool
	wf := func(p string, i os.FileInfo, err error) error {
		if p == "/dev/disk/by-uuid/" {
			return err
		}
		for _, y := range devs {
			tgt, err := os.Readlink(p)
			if err != nil {
				return err
			}
			if filepath.Base(y) == filepath.Base(tgt) && i.Name() == uuid {
				found = true
			}
		}
		return err
	}
	err = filepath.Walk("/dev/disk/by-uuid/", wf)
	if err != nil {
		return false, err
	}
	return found, err
}

// ensureDeviceIsMounted will first check if the device d is mounted. If not, then the user will be prompted to mount the device.
func ensureDeviceIsMounted(d core.Device) error {
	firstCheck := true
	for {
		m, err := deviceIsMountedByUUID(d.MountPoint, d.UUID)
		if err != nil {
			return err
		}
		if !m {
			if !firstCheck {
				log.Errorf("Device UUID=%s not found!", d.UUID)
			}
			// log.WithFields(logrus.Fields{
			// "deviceName":       d.Name,
			// "deviceMountPoint": d.MountPoint,
			// }).Printf("Mount %q then press the Enter key to continue...", d.Name)
			// var input string
			// fmt.Scanln(&input)
			firstCheck = false
		} else {
			break
		}
	}
	return nil
}
