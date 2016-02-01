package core

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"
)

func NewID() (string, error) {
	// Got this from somewhere on stackoverflow
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80

	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// sha1sum gets the sha1 hash of filePath using an external hashing tool.
func sha1sum(filePath string) (hash string, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		err = fmt.Errorf("sha1sum: %s", err.Error())
		return
	}
	sh := sha1.New()
	_, err = io.Copy(sh, f)
	if err != nil {
		err = fmt.Errorf("sha1sum: %s", err.Error())
		return
	}
	hash = hex.EncodeToString(sh.Sum(nil))
	return
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
