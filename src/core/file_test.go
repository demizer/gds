package core

import "testing"

type expectDevice struct {
	name      string
	usedBytes uint64
}

func checkDevices(t *testing.T, c *Context, e []expectDevice) {
	expectDeviceByName := func(n string) *expectDevice {
		for _, x := range e {
			if x.name == n {
				return &x
			}
		}
		return nil
	}
	dNum := 0
	lastDevice := false
	inTolerance := false
	for _, xy := range c.Devices {
		u, _ := c.Devices.DeviceByName(xy.Name)
		if dNum == len(c.Devices)-1 {
			lastDevice = true
			// The last device can fluctuate in size due to the sync context data file being stored on it.
			inTolerance = (u.SizeWritn > expectDeviceByName(xy.Name).usedBytes-50 &&
				u.SizeWritn < expectDeviceByName(xy.Name).usedBytes+50)
		}
		if (u.SizeWritn != expectDeviceByName(xy.Name).usedBytes) || (lastDevice && !inTolerance) {
			t.Errorf("MountPoint: %q\n\t Got Used Bytes: %d Expect: %d\n",
				xy.MountPoint, u.SizeWritn, expectDeviceByName(xy.Name).usedBytes)
		}
		dNum++
	}
}

func TestGetFileByName(t *testing.T) {
	f := &FileIndex{
		&File{Name: "test1"},
		&File{Name: "test2"},
	}
	_, err := f.FileByName("test3")
	if d, ok := err.(*FileNotFoundError); !ok {
		t.Errorf("Expect: %T Got: %T", new(FileNotFoundError), d)
	}
	if new(FileNotFoundError).Error() == "" {
		t.Error("Missing error message")
	}
}
