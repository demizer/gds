package core

import (
	"io/ioutil"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// NewMountPoint creates a new test mountpoint and returns a string with the directory ready for usage. Any errors creating
// the mountpoint wil Fail the test.
func NewMountPoint(t *testing.T, basePath string, prefix string) string {
	p, err := ioutil.TempDir(basePath, prefix)
	if err != nil {
		t.Fatalf("EXPECT: path to temp mount GOT: %s", err)
	}
	return p
}

func TestSha1Sum(t *testing.T) {
	_, err := sha1sum("/root")
	if err == nil {
		t.Error("EXPECT: error permission denied GOT: No errors")
	}
}

func TestLUtimesNano(t *testing.T) {
	f := File{
		ModTime: time.Now(),
	}
	mTimeval := syscall.NsecToTimespec(f.ModTime.UnixNano())
	times := []syscall.Timespec{
		mTimeval,
		mTimeval,
	}
	err := LUtimesNano("/root", times)
	if err == nil {
		t.Errorf("Expect: Error Got: %q", err)
	}
	err = LUtimesNano("/root\x00", times)
	if err == nil {
		t.Errorf("Expect: Error Got: %q", err)
	}
}

// mountTestDevice mounts a device to mountPoint for testing. The user running the test must have permission to mount the
// device. The following line needs to be added to /etc/fstab
//
// /home/<%USER%>/src/gds/testdata/filesystems/td-1-ext4 /mnt/gds-test ext4 noauto,defaults,user 0 0
//
// cmd can be the commands "mount" or "unmount".
func mountTestDevice(cmd string, mountPoint string) error {
	c := exec.Command(cmd, mountPoint)
	err := c.Run()
	if err != nil {
		return err
	}
	return nil
}

// TestDeviceIsMountedByUUID tests for device mount detection by UUID. This requires that the mount point "/mnt/gds-test" be
// mountable by the user running the tests.
func TestDeviceIsMountedByUUID(t *testing.T) {
	mountTestDevice("mount", "testdata/filesystems/td-1-ext4")
	b, err := deviceIsMountedByUUID("/mnt/gds-test", "127b7cc4-9c16-4d1d-8125-a51d668cf6df")
	if err != nil {
		t.Error("Error mounting test device %q: %s", "testdata/filesystems/td-1-ext4", err)
	}
	if !b {
		t.Error("EXPECT: testdata/filesystems/td-1-ext4 (uuid 127b7cc4-9c16-4d1d-8125-a51d668cf6df) is mounted\n  " +
			"GOT: Device not mounted")
	}
	mountTestDevice("umount", "testdata/filesystems/td-1-ext4")
}

func TestDeviceIsMountedByUUID2(t *testing.T) {
	f := &syncTest{
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       936960,
					MountPoint: "/mnt/gds-test",
				},
			}
		},
	}
	dl := f.deviceList()
	d, _ := dl.DeviceByName("Test Device 0")
	mountTestDevice("mount", "testdata/filesystems/td-1-ext4")
	if _, err := d.IsMounted(); err != nil {
		t.Errorf("EXPECT: testdata/filesystems/td-1-ext4 (uuid 127b7cc4-9c16-4d1d-8125-a51d668cf6df) is mounted\n  "+
			"GOT: Error %s", err)
	}
	mountTestDevice("umount", "testdata/filesystems/td-1-ext4")
}
