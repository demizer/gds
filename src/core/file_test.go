package core

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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
	// All tests will be saved to test_temp_dir instead of "/tmp". Saving test output to "/tmp" directory can cause
	// problems with testing if "/tmp" is mounted to memory. The Kernel reclaims as much space as possible, this causes
	// directory sizes to behave differently when files are removed from the directory. In a normal filesystem, the
	// directory sizes are unchanged after files are removed from the directory, but in a RAM mounted /tmp, the directory
	// sizes are reclaimed immediately.
	test_temp_dir = func() string {
		cdir, _ := os.Getwd()
		return path.Clean(path.Join(cdir, "..", "..", "testdata", "temp"))
	}()
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

// fileTests test subdirectory creation, fileinfo synchronization, and file duplication.
type fileSyncTest struct {
	outputStreams int
	backupPath    string
	fileList      func() FileList // Must come before deviceList in the anon struct
	deviceList    func() DeviceList
	catalog       func() Catalog
	expectErrors  func() []error
	splitMinSize  uint64
}

func runFileSyncTest(t *testing.T, f *fileSyncTest) {
	c := NewContext()
	var err error
	c.BackupPath = f.backupPath
	if f.fileList == nil {
		c.Files, err = NewFileList(c)
		if err != nil {
			t.Error(err)
			return
		}
	} else {
		c.Files = f.fileList()
	}
	c.Devices = f.deviceList()
	c.OutputStreamNum = f.outputStreams
	c.SplitMinSize = f.splitMinSize
	c.Catalog = NewCatalog(c)
	// spd.Dump(c.Catalog)
	// os.Exit(1)

	// Do the work!
	err2 := Sync(c)
	if len(err2) != 0 {
		found := false
		if f.expectErrors == nil {
			t.Errorf("Expect: No errors\n\t  Got: %s", spd.Sprint(err2))
			return
		}
		for _, e := range err2 {
			for _, e2 := range f.expectErrors() {
				if e.Error() == e2.Error() {
					found = true
					break
				}
			}
			if found {
				continue
			}
			t.Error(err)
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
					t.Error(err)
				}
				if cvf.SrcSha1 != sum {
					t.Errorf("Error: %s\n",
						fmt.Errorf("File: %q SrcSha1: %q, DestSha1: %q", cvf.Name, cvf.SrcSha1, sum))
				}
			}
			// Check uid, gid, and mod time
			fi, err := os.Lstat(cvf.DestPath)
			if err != nil {
				t.Error(err)
				continue
			}
			if fi.Mode() != cvf.Mode {
				t.Errorf("File: %q\n\t  Got Mode: %q Expect: %q\n", cvf.Name, fi.Mode(), cvf.Mode)
			}
			if fi.ModTime() != cvf.ModTime {
				t.Errorf("File: %q\n\t  Got ModTime: %q Expect: %q\n", cvf.Name, fi.ModTime(), cvf.ModTime)
			}
			if int(fi.Sys().(*syscall.Stat_t).Uid) != cvf.Owner {
				t.Errorf("File: %q\n\t  Got Owner: %q Expect: %q\n",
					cvf.ModTime, int(fi.Sys().(*syscall.Stat_t).Uid), cvf.Owner)
			}
			if int(fi.Sys().(*syscall.Stat_t).Gid) != cvf.Group {
				t.Errorf("File: %q\n\t  Got Group: %d Expect: %d\n",
					cvf.Name, int(fi.Sys().(*syscall.Stat_t).Gid), cvf.Group)
			}
			// Check the size of the output file
			ls, err := os.Lstat(cvf.DestPath)
			if err != nil {
				t.Errorf("Could not stat %q, %s", cvf.DestPath, err.Error())
			}
			if cvf.SplitEndByte == 0 && uint64(ls.Size()) != cvf.Size {
				t.Errorf("File: %q\n\t  Got Size: %d Expect: %d\n",
					cvf.DestPath, ls.Size, cvf.Size)
			} else if cvf.SplitEndByte != 0 && uint64(ls.Size()) != cvf.SplitEndByte-cvf.SplitStartByte {
				t.Errorf("File: %q\n\t  Got Size: %d Expect: %d\n",
					cvf.DestPath, ls.Size, cvf.SplitEndByte-cvf.SplitStartByte)
			}

		}
		// Check the size of the MountPoint
		dev := c.Devices.GetDeviceByName(cx)
		ms, err := checkMountpointUsage(dev.MountPoint)
		if err != nil {
			t.Error(err)
		}
		log.Info("Mountpoint used size", "size", ms)
		log.Info("Device used size", "size", dev.UsedSize)
		if uint64(ms) != dev.UsedSize {
			t.Errorf("MountPoint: %q\n\t  Got Size: %d Expect: %d\n", dev.MountPoint, ms, dev.UsedSize)
		}
	}
}

func TestFileSyncSimpleCopy(t *testing.T) {
	f := &fileSyncTest{
		backupPath: "../../testdata/filesync_test01_freebooks/",
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir(test_temp_dir, "mountpoint-0-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  28173338480,
					MountPoint: tmp0,
				},
			)
			return n
		},
	}
	runFileSyncTest(t, f)
}

func TestFileSyncPerms(t *testing.T) {
	f := &fileSyncTest{
		backupPath: "/dev/null/",
		fileList: func() FileList {
			var n FileList
			test_output_dir, _ = ioutil.TempDir(test_temp_dir, "mountpoint-0-")
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
	}
	runFileSyncTest(t, f)
}

func TestFileSyncSubDirs(t *testing.T) {
	f := &fileSyncTest{
		backupPath: "../../testdata/filesync_test03_subdirs/",
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir(test_temp_dir, "mountpoint-0-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  28173338480,
					MountPoint: tmp0,
				},
			)
			return n
		},
	}
	runFileSyncTest(t, f)
}

func TestFileSyncSymlinks(t *testing.T) {
	f := &fileSyncTest{
		backupPath: "../../testdata/filesync_test04_symlinks/",
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir(test_temp_dir, "mountpoint-0-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  28173338480,
					MountPoint: tmp0,
				},
			)
			return n
		},
	}
	runFileSyncTest(t, f)
}

func TestFileSyncBackupathIncluded(t *testing.T) {
	f := &fileSyncTest{
		backupPath: "../../testdata/filesync_test01_freebooks",
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir(test_temp_dir, "mountpoint-0-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  28173338480,
					MountPoint: tmp0,
				},
			)
			return n
		},
	}
	runFileSyncTest(t, f)
}

func TestFileSyncFileSplitAcrossDevices(t *testing.T) {
	f := &fileSyncTest{
		backupPath:   "../../testdata/filesync_test01_freebooks",
		splitMinSize: 1000,
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir(test_temp_dir, "mountpoint-0-")
			tmp1, _ := ioutil.TempDir(test_temp_dir, "mountpoint-1-")
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
	}
	runFileSyncTest(t, f)
}

func TestFileSyncLargeFileAcrossOneWholeDeviceAndHalfAnother(t *testing.T) {
	f := &fileSyncTest{
		backupPath:   "../../testdata/filesync_test07_file_split/",
		splitMinSize: 1000,
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir(test_temp_dir, "mountpoint-0-")
			tmp1, _ := ioutil.TempDir(test_temp_dir, "mountpoint-1-")
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
	}
	runFileSyncTest(t, f)
}

func TestFileSyncLargeFileAcrossThreeDevices(t *testing.T) {
	f := &fileSyncTest{
		backupPath: "../../testdata/filesync_test07_file_split",
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir(test_temp_dir, "mountpoint-0-")
			tmp1, _ := ioutil.TempDir(test_temp_dir, "mountpoint-1-")
			tmp2, _ := ioutil.TempDir(test_temp_dir, "mountpoint-2-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  3495255,
					MountPoint: tmp0,
				},
				Device{
					Name:       "Test Device 1",
					SizeBytes:  3495255,
					MountPoint: tmp1,
				},
				Device{
					Name:       "Test Device 2",
					SizeBytes:  3495250,
					MountPoint: tmp2,
				},
			)
			return n
		},
	}
	runFileSyncTest(t, f)
}

// TestFileSyncDirsWithLotsOfFiles checks syncing directories with thousands of files and directories that _had_ thousands of
// files. Directories that had thousands of files are still allocated in the filesystem as containing thousands of file
// pointers, but since filesystems don't reclaim space, recreating these directories on the destination drive will allocate
// the blocksize of the device--4096 bytes.
func TestFileSyncDirsWithLotsOfFiles(t *testing.T) {
	test_temp_dir, err := ioutil.TempDir(test_temp_dir, "gds-filetests-")
	if err != nil {
		t.Error(err)
	}
	// Copy filesync_test09_directories to the temp dir and delete all of the files in the dir
	cmd := exec.Command("/usr/bin/cp", "-R", "../../testdata/filesync_test09_directories", test_temp_dir)
	err = cmd.Run()
	if err != nil {
		t.Error(err)
	}
	// Duplicate the sub dir
	cmd = exec.Command("/usr/bin/cp", "-R", path.Join(test_temp_dir, "filesync_test09_directories", "dir_with_thousand_files"),
		path.Join(test_temp_dir, "filesync_test09_directories", "dir_with_thousand_files_deleted"))
	err = cmd.Run()
	if err != nil {
		t.Error(err)
	}
	// Delete all of the files in the directory
	files, err := filepath.Glob(path.Join(test_temp_dir, "filesync_test09_directories", "dir_with_thousand_files_deleted", "*"))
	if err != nil {
		t.Error(err)
	}
	for _, y := range files {
		os.Remove(y)
	}

	f := &fileSyncTest{
		backupPath: path.Join(test_temp_dir, "filesync_test09_directories"),
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir(test_temp_dir, "mountpoint-0-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  4300000,
					MountPoint: tmp0,
				},
			)
			return n
		},
	}
	runFileSyncTest(t, f)
	fmt.Println("TestFileSyncDirsWithLotsOfFiles output directory:", test_temp_dir)
}
