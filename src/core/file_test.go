package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

type file struct {
	Path      string
	Size      string
	SizeBytes uint64
}

var (
	// Used for tests that expect errors
	test_output_dir string
)

// fileTests test subdirectory creation, fileinfo synchronization, and file duplication.
var fileTests = [...]struct {
	testName      string
	outputStreams int
	backupPath    string
	deviceList    func() DeviceList
	catalog       func() Catalog
	expectErrors  func() []error
}{
	{
		testName: "Test #1 - Simple Copy",
		// ../../testdata/testwalk_001/ should be ommitted from all output
		backupPath: "../../testdata/filesync_test01_freebooks/",
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir("", "gds-filetests-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  28173338480,
					MountPoint: tmp0,
				},
			)
			return n
		},
	},
	{
		testName:   "Test #2 - Permissions",
		backupPath: "../../testdata/filesync_test02_permissions/",
		deviceList: func() DeviceList {
			var n DeviceList
			test_output_dir, _ = ioutil.TempDir("", "gds-filetests-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  28173338480,
					MountPoint: test_output_dir,
				},
			)
			return n
		},
		expectErrors: func() []error {
			var e []error
			e = append(e, SyncIncorrectOwnershipError{
				FilePath: filepath.Join(test_output_dir, "diff_user"),
				OwnerId:  25755,
				UserId:   os.Getuid(),
			}, SyncIncorrectOwnershipError{
				FilePath: filepath.Join(test_output_dir, "diff_user_unreadable"),
				OwnerId:  25755,
				UserId:   os.Getuid(),
			})
			return e
		},
	},
	{
		testName:   "Test #3 - Subdirs",
		backupPath: "../../testdata/filesync_test03_subdirs/",
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir("", "gds-filetests-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  28173338480,
					MountPoint: tmp0,
				},
			)
			return n
		},
	},
	{
		testName:   "Test #4 - Symlinks",
		backupPath: "../../testdata/filesync_test04_symlinks/",
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir("", "gds-filetests-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  28173338480,
					MountPoint: tmp0,
				},
			)
			return n
		},
	},
	{
		testName:   "Test #5 - Copy backup directory with contents",
		backupPath: "../../testdata/filesync_test01_freebooks",
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir("", "gds-filetests-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  28173338480,
					MountPoint: tmp0,
				},
			)
			return n
		},
	},
}

func TestFileSync(t *testing.T) {
	for _, y := range fileTests {
		c := NewContext()
		var err error
		c.Files, err = NewFileList(y.backupPath)
		if err != nil {
			t.Errorf("Test: %q\n\t  Error: %q\n", y.testName, err.Error())
			return
		}
		c.Devices = y.deviceList()
		c.OutputStreamNum = y.outputStreams
		c.Catalog = NewCatalog(y.backupPath, c.Devices, &c.Files)

		// Do the work!
		err2 := Sync(c)
		if len(err2) != 0 {
			found := false
			if y.expectErrors == nil {
				t.Errorf("Test: %q\n\t  Expect: No errors\n\t  Got: %s", y.testName, spd.Sprint(err2))
				continue
			}
			for _, e := range err2 {
				for _, e2 := range y.expectErrors() {
					if e == e2 {
						found = true
						break
					}
				}
				if found {
					continue
				}
				t.Errorf("Test: %q\n\t  Error: %s\n", y.testName, e.Error())
			}
		}

		// Check the work!
		for _, cv := range c.Catalog {
			for _, cvf := range cv {
				if cvf.FileType == DIRECTORY || cvf.Owner != os.Getuid() {
					continue
				}
				if cvf.FileType != DIRECTORY && cvf.FileType != SYMLINK {
					sum, err := sha1sum(cvf.DestPath)
					if err != nil {
						t.Errorf("Test: %q\n\t  Error: %s\n", y.testName, err.Error())
					}
					if cvf.SrcSha1 != sum {
						t.Errorf("Test: %q\n\t  Error: %s\n", y.testName,
							fmt.Errorf("File: %q SrcSha1: %q, DestSha1: %q", cvf.Name, cvf.SrcSha1, sum))
					}
				}
				// Check uid, gid, and mod time
				fi, err := os.Lstat(cvf.DestPath)
				if err != nil {
					t.Errorf("Test: %q\n\t  Error: %s\n", y.testName, err)
					continue
				}
				if fi.Mode() != cvf.Mode {
					t.Errorf("Test: %q\n\t  File: %q\n\t  Got Mode: %q Expect: %q\n",
						y.testName, cvf.Name, fi.Mode(), cvf.Mode)
				}
				if fi.ModTime() != cvf.ModTime {
					t.Errorf("Test: %q\n\t  File: %q\n\t  Got ModTime: %q Expect: %q\n",
						y.testName, cvf.Name, fi.ModTime(), cvf.ModTime)
				}
				if int(fi.Sys().(*syscall.Stat_t).Uid) != cvf.Owner {
					t.Errorf("Test: %q\n\t  File: %q\n\t  Got Owner: %q Expect: %q\n",
						y.testName, cvf.ModTime, int(fi.Sys().(*syscall.Stat_t).Uid), cvf.Owner)
				}
				if int(fi.Sys().(*syscall.Stat_t).Gid) != cvf.Group {
					t.Errorf("Test: %q\n\t  File: %q\n\t  Got Group: %q Expect: %q\n",
						y.testName, cvf.Name, int(fi.Sys().(*syscall.Stat_t).Gid), cvf.Group)
				}
			}
		}
	}
}

// A test for checking the error value of checkDevicePoolSpace()
func TestCheckDevicePoolSpace(t *testing.T) {
	file := File{
		Name: "Large File",
		Size: 100000,
	}
	device := Device{
		Name:      "Device 0",
		SizeBytes: 10000,
	}
	var fl FileList
	fl = append(fl, file)
	var dl DeviceList
	dl = append(dl, device)
	eerr := NotEnoughStorageSpaceError{100000, 10000}
	err := checkDevicePoolSpace(fl, dl)
	if err != eerr {
		t.Errorf("Test: NotEnoughStorageSpaceError Check\n\t  Got: %q Expect: %q\n", err, eerr)
	}
}
