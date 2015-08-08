package core

import (
	"os"
	"path/filepath"
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

// checkMountpointUsage calculates the total size of files located under the mountpoint (m).
func checkMountpointUsage(m string) (int64, error) {
	var byts int64 = 0
	walkFunc := func(p string, i os.FileInfo, err error) error {
		if p == m {
			return nil
		}
		byts += i.Size()
		return nil
	}
	err := filepath.Walk(m, walkFunc)
	return byts, err
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
	for _, xy := range c.Devices {
		u, _ := c.Devices.DeviceByName(xy.Name)
		if u.UsedSize != expectDeviceByName(xy.Name).usedBytes {
			t.Errorf("MountPoint: %q\n\t Got Used Bytes: %d Expect: %d\n",
				xy.MountPoint, u.UsedSize, expectDeviceByName(xy.Name).usedBytes)
		}
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
