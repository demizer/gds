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
	test2_output_dir string
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
			test2_output_dir, _ = ioutil.TempDir("", "gds-filetests-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  28173338480,
					MountPoint: test2_output_dir,
				},
			)
			return n
		},
		expectErrors: func() []error {
			var e []error
			e = append(e, SyncIncorrectOwnershipError{
				FilePath: filepath.Join(test2_output_dir, "diff_user"),
				OwnerId:  25755,
				UserId:   os.Getuid(),
			}, SyncIncorrectOwnershipError{
				FilePath: filepath.Join(test2_output_dir, "diff_user_unreadable"),
				OwnerId:  25755,
				UserId:   os.Getuid(),
			})
			return e
		},
	},
}

func TestFileSync(t *testing.T) {
	for _, y := range fileTests {
		c := NewContext()
		fmt.Println("Test:", y.testName, "- START")
		var err error
		c.Files, err = NewFileList(y.backupPath)
		if err != nil {
			t.Errorf("Test: %q\n\t Error: %q\n", y.testName, err.Error())
			return
		}
		c.Devices = y.deviceList()
		c.OutputStreamNum = y.outputStreams
		c.Catalog = NewCatalog(y.backupPath, c.Devices, &c.Files)

		// Do the work!
		err2 := Sync(c)
		if len(err2) != 0 {
			found := false
			for _, e := range err2 {
				for _, e2 := range y.expectErrors() {
					if e.Error() == e2.Error() {
						found = true
						break
					}
				}
				if found {
					continue
				}
				t.Errorf("Test: %q\n\t Error: %s\n", y.testName, e.Error())
			}
		}

		// Check the work!
		for _, cv := range c.Catalog {
			for _, cvf := range cv {
				if cvf.IsDir || cvf.Owner != os.Getuid() {
					continue
				}
				sum, err := sha1sum(cvf.DestPath)
				if err != nil {
					t.Errorf("Test: %q\n\t Error: %s\n", y.testName, err.Error())
				}
				if cvf.SrcSha1 != sum {
					t.Errorf("Test: %q\n\t Error: %s\n", y.testName,
						fmt.Errorf("SrcSha1: %q, DestSha1: %q", cvf.SrcSha1, sum))
				}
				// Check uid, gid, and mod time
				f, err := os.Open(cvf.DestPath)
				if err != nil {
					t.Errorf("Test: %q\n\t Error: %s\n", y.testName, err)
				}
				fs, err := f.Stat()
				if err != nil {
					t.Errorf("Test: %q\n\t Error: %s\n", y.testName, err)
				}
				if fs.Mode() != cvf.Mode {
					t.Errorf("Test: %q\n\t Got Mode: %q Expect: %q\n", y.testName, fs.Mode(), cvf.Mode)
				}
				if fs.ModTime() != cvf.ModTime {
					t.Errorf("Test: %q\n\t Got ModTime: %q Expect: %q\n", y.testName, fs.ModTime(), cvf.ModTime)
				}
				if int(fs.Sys().(*syscall.Stat_t).Uid) != cvf.Owner {
					t.Errorf("Test: %q\n\t Got Owner: %q Expect: %q\n", y.testName, int(fs.Sys().(*syscall.Stat_t).Uid), cvf.Owner)
				}
				if int(fs.Sys().(*syscall.Stat_t).Gid) != cvf.Group {
					t.Errorf("Test: %q\n\t Got Group: %q Expect: %q\n", y.testName, int(fs.Sys().(*syscall.Stat_t).Gid), cvf.Group)
				}
			}
		}
		pf := "PASS"
		if t.Failed() {
			pf = "FAIL"
		}
		fmt.Println("Test:", y.testName, "-", pf)
	}
}
