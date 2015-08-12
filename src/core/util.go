package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
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

// From https://github.com/docker/docker/blob/master/pkg/system/utimes_linux.go
func LUtimesNano(path string, ts []syscall.Timespec) error {
	// These are not currently available in syscall
	AT_FDCWD := -100
	AT_SYMLINK_NOFOLLOW := 0x100

	var _path *byte
	_path, err := syscall.BytePtrFromString(path)
	if err != nil {
		return err
	}

	if _, _, err := syscall.Syscall6(syscall.SYS_UTIMENSAT, uintptr(AT_FDCWD),
		uintptr(unsafe.Pointer(_path)), uintptr(unsafe.Pointer(&ts[0])),
		uintptr(AT_SYMLINK_NOFOLLOW), 0, 0); err != 0 && err != syscall.ENOSYS {
		return err
	}

	return nil
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
