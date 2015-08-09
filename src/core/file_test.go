package core

import (
	"os"
	"testing"
)

func TestBadFileMetadataError(t *testing.T) {
	if new(BadFileMetadatError).Error() == "" {
		t.Error("Missing error message")
	}
}

func TestGetFileByName(t *testing.T) {
	f := &FileList{
		File{Name: "test1"},
		File{Name: "test2"},
	}
	_, err := f.FileByName("test3")
	if d, ok := err.(*FileNotFound); !ok {
		t.Errorf("Expect: %T Got: %T", new(FileNotFound), d)
	}
	if new(FileNotFound).Error() == "" {
		t.Error("Missing error message")
	}
}

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
		if dNum+1 == len(c.Devices) {
			lastDevice = true
			// The last device can fluctuate in size due to the sync context data file being stored on it.
			inTolerance = (u.UsedSize < expectDeviceByName(xy.Name).usedBytes-50 &&
				u.UsedSize > expectDeviceByName(xy.Name).usedBytes+50)
		}
		if (u.UsedSize != expectDeviceByName(xy.Name).usedBytes && !lastDevice) || (lastDevice && inTolerance) {
			t.Errorf("MountPoint: %q\n\t Got Used Bytes: %d Expect: %d\n",
				xy.MountPoint, u.UsedSize, expectDeviceByName(xy.Name).usedBytes)
		}
		dNum++
	}
}

func TestDestPathSha1SumError(t *testing.T) {
	if new(BadDestPathSha1Sum).Error() == "" {
		t.Error("Missing error message")
	}
}

func TestDestPathSha1SumCopyError(t *testing.T) {
	f := File{}
	_, err := f.DestPathSha1Sum()
	if err == nil {
		t.Error("Expect: Error Got: No Error")
	}
}

func TestDestPathSha1Sum(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}
	expectSha1 := "08cdd7178a20032c27d152a1f440334ee5f132a0"

	// Check bad dest path
	f := File{
		Name:    "alice",
		SrcSha1: expectSha1,
	}
	_, err := f.DestPathSha1Sum()
	if _, ok := err.(*os.PathError); !ok {
		t.Errorf("Expect: *os.PathError Got: %T", err)
		return
	}

	// Check no error
	f.DestPath = "../../testdata/filesync_freebooks/alice/alice_in_wonderland_by_lewis_carroll_gutenberg.org.htm"
	s, err := f.DestPathSha1Sum()
	if err != nil {
		t.Errorf("Expect: No error Got: %T (%q)", err, err.Error())
		return
	}

	f.DestPath = "../../testdata/filesync_freebooks/ulysses/ulysses_by_james_joyce_gutenberg.org.htm"
	s, err = f.DestPathSha1Sum()
	// Check sha1
	if s == expectSha1 {
		t.Error("Bad sha1 sum")
		return
	}
}

func TestNewFileList(t *testing.T) {
	c := NewContext("/root")
	_, err := NewFileList(c)
	if err == nil {
		t.Error("Expect: Error  Got: No errors")
	}
}
