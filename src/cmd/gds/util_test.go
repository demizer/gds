package main

import (
	"os/exec"
	"testing"
	"time"
)

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
	err := mountTestDevice("mount", "/mnt/gds-test")
	time.Sleep(time.Second)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: Error: %s", err)
	}
	b, err := deviceIsMountedByUUID("/mnt/gds-test", "127b7cc4-9c16-4d1d-8125-a51d668cf6df")
	if err != nil {
		t.Error("Error mounting test device %q: %s", "testdata/filesystems/td-1-ext4", err)
	}
	if !b {
		t.Error("EXPECT: testdata/filesystems/td-1-ext4 (uuid 127b7cc4-9c16-4d1d-8125-a51d668cf6df) is mounted\n  " +
			"GOT: Device not mounted")
	}
	mountTestDevice("umount", "/mnt/gds-test")
}
