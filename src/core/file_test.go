package core

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	log "gopkg.in/inconshreveable/log15.v2"
)

func init() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "Enable debug output.")
	flag.Parse()

	if !debug {
		log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StdoutHandler))
	}
}

type file struct {
	Path      string
	Size      string
	SizeBytes uint64
}

var (
	// Used for tests that expect errors
	test_output_dir string
)

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

// fileTests test subdirectory creation, fileinfo synchronization, and file duplication.
var fileTests = [...]struct {
	testName      string
	outputStreams int
	backupPath    string
	fileList      func() FileList // Must come before deviceList in the anon struct
	deviceList    func() DeviceList
	catalog       func() Catalog
	expectErrors  func() []error
	splitMinSize  uint64
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
		backupPath: "/dev/null/",
		fileList: func() FileList {
			var n FileList
			test_output_dir, _ = ioutil.TempDir("", "gds-filetests-")
			n = append(n,
				File{
					Name:     "diff_user",
					FileType: FILE,
					Size:     1024,
					Path:     "/dev/zero",
					DestPath: path.Join(test_output_dir, "diff_user"),
					Mode:     0640,
					ModTime:  time.Now(),
					Owner:    55000,
					Group:    55000,
				},
				File{
					Name:     "script.sh",
					FileType: FILE,
					Size:     1024,
					Path:     "/dev/zero",
					DestPath: path.Join(test_output_dir, "script.sh"),
					Mode:     0777,
					ModTime:  time.Now(),
					Owner:    os.Getuid(),
					Group:    os.Getgid(),
				},
				File{
					Name:     "some_dir",
					Path:     "/dev/zero",
					FileType: DIRECTORY,
					Size:     4096,
					DestPath: path.Join(test_output_dir, "some_dir"),
					Mode:     0755,
					ModTime:  time.Now(),
					Owner:    os.Getuid(),
					Group:    55000,
				},
			)
			return n
		},
		deviceList: func() DeviceList {
			var n DeviceList
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
				OwnerId:  55000,
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
	{
		testName:     "Test #6 - Split file across devices",
		backupPath:   "../../testdata/filesync_test01_freebooks",
		splitMinSize: 1000,
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir("", "gds-filetests-")
			tmp1, _ := ioutil.TempDir("", "gds-filetests-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  850000,
					MountPoint: tmp0,
				},
				Device{
					Name:       "Test Device 1",
					SizeBytes:  850000,
					MountPoint: tmp1,
				},
			)
			return n
		},
	},
	{
		testName:     "Test #7 - Large file on one whole device and partly on another",
		backupPath:   "../../testdata/filesync_test07_file_split/",
		splitMinSize: 1000,
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir("", "gds-filetests-")
			tmp1, _ := ioutil.TempDir("", "gds-filetests-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  9999999,
					MountPoint: tmp0,
				},
				Device{
					Name:       "Test Device 1",
					SizeBytes:  850000,
					MountPoint: tmp1,
				},
			)
			return n
		},
	},
}

// TestFileSync is the main test function for testing file sync operations. Being it is the main test function, it is a
// little large...
func TestFileSync(t *testing.T) {
	for _, y := range fileTests {
		fmt.Println("\n--- Running test: ", y.testName, "\n")
		c := NewContext()
		var err error
		c.BackupPath = y.backupPath
		if y.fileList == nil {
			c.Files, err = NewFileList(c)
			if err != nil {
				t.Errorf("%s\n\t  Error: %s\n", y.testName, err.Error())
				return
			}
		} else {
			c.Files = y.fileList()
		}
		c.Devices = y.deviceList()
		c.OutputStreamNum = y.outputStreams
		c.SplitMinSize = y.splitMinSize
		c.Catalog = NewCatalog(c)

		// Do the work!
		err2 := Sync(c)
		if len(err2) != 0 {
			found := false
			if y.expectErrors == nil {
				t.Errorf("%s\n\t  Expect: No errors\n\t  Got: %s", y.testName, spd.Sprint(err2))
				continue
			}
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
				t.Errorf("%s\n\t  Error: %s\n", y.testName, e.Error())
			}
		}

		// Check the work!
		for cx, cv := range c.Catalog {
			for _, cvf := range cv {
				if cvf.FileType == DIRECTORY || cvf.Owner != os.Getuid() {
					continue
				}
				if cvf.FileType != DIRECTORY && cvf.FileType != SYMLINK {
					sum, err := sha1sum(cvf.DestPath)
					if err != nil {
						t.Errorf("%s\n\t  Error: %s\n", y.testName, err.Error())
					}
					if cvf.SrcSha1 != sum {
						t.Errorf("%s\n\t  Error: %s\n", y.testName,
							fmt.Errorf("File: %q SrcSha1: %q, DestSha1: %q", cvf.Name, cvf.SrcSha1, sum))
					}
				}
				// Check uid, gid, and mod time
				fi, err := os.Lstat(cvf.DestPath)
				if err != nil {
					t.Errorf("%s\n\t  Error: %s\n", y.testName, err)
					continue
				}
				if fi.Mode() != cvf.Mode {
					t.Errorf("%s\n\t  File: %q\n\t  Got Mode: %q Expect: %q\n",
						y.testName, cvf.Name, fi.Mode(), cvf.Mode)
				}
				if fi.ModTime() != cvf.ModTime {
					t.Errorf("%s\n\t  File: %q\n\t  Got ModTime: %q Expect: %q\n",
						y.testName, cvf.Name, fi.ModTime(), cvf.ModTime)
				}
				if int(fi.Sys().(*syscall.Stat_t).Uid) != cvf.Owner {
					t.Errorf("%s\n\t  File: %q\n\t  Got Owner: %q Expect: %q\n",
						y.testName, cvf.ModTime, int(fi.Sys().(*syscall.Stat_t).Uid), cvf.Owner)
				}
				if int(fi.Sys().(*syscall.Stat_t).Gid) != cvf.Group {
					t.Errorf("%s\n\t  File: %q\n\t  Got Group: %d Expect: %d\n",
						y.testName, cvf.Name, int(fi.Sys().(*syscall.Stat_t).Gid), cvf.Group)
				}
				// Check the size of the output file
				ls, err := os.Lstat(cvf.DestPath)
				if err != nil {
					t.Errorf("Could not stat %q, %s", cvf.DestPath, err.Error())
				}
				if cvf.SplitEndByte == 0 && uint64(ls.Size()) != cvf.Size {
					t.Errorf("%s\n\t  File: %q\n\t  Got Size: %d Expect: %d\n",
						y.testName, cvf.DestPath, ls.Size, cvf.Size)
				} else if cvf.SplitEndByte != 0 && uint64(ls.Size()) != cvf.SplitEndByte-cvf.SplitStartByte {
					t.Errorf("%s\n\t  File: %q\n\t  Got Size: %d Expect: %d\n",
						y.testName, cvf.DestPath, ls.Size, cvf.SplitEndByte-cvf.SplitStartByte)
				}

			}
			// Check the size of the MountPoint
			dev := c.Devices.GetDeviceByName(cx)
			ms, err := checkMountpointUsage(dev.MountPoint)
			if err != nil {
				t.Error(err)
			}
			if uint64(ms) != dev.UsedSize {
				t.Errorf("%s\n\t  MountPoint: %q\n\t  Got Size: %d Expect: %d\n",
					y.testName, dev.MountPoint, ms, dev.UsedSize)
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
