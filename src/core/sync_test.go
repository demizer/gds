package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
)

var (
	// All tests will be saved to testTempDir instead of "/tmp". Saving test output to "/tmp" directory can cause
	// problems with testing if "/tmp" is mounted to memory. The Kernel reclaims as much space as possible, this causes
	// directory sizes to behave differently when files are removed from the directory. In a normal filesystem, the
	// directory sizes are unchanged after files are removed from the directory, but in a RAM mounted /tmp, the directory
	// sizes are reclaimed immediately.
	testTempDir = func() string {
		cdir, _ := os.Getwd()
		return path.Clean(path.Join(cdir, "..", "..", "testdata", "temp"))
	}()
)

// fileTests test subdirectory creation, fileinfo synchronization, and file duplication.
type syncTest struct {
	outputStreams     int
	backupPath        string
	fileList          func() FileList // Must come before deviceList in the anon struct
	deviceList        func() DeviceList
	catalog           func() Catalog
	expectErrors      func() []error
	expectDeviceUsage func() []expectDevice
	splitMinSize      uint64
}

func runSyncTest(t *testing.T, f *syncTest) *Context {
	c := NewContext(f.backupPath)
	var err error
	if f.fileList == nil {
		c.Files, err = NewFileList(c)
		if err != nil {
			t.Error(err)
			return nil
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
			return nil
		}
		for _, e := range err2 {
			for _, e2 := range f.expectErrors() {
				if e.Error() == e2.Error() {
					found = true
				} else {
					t.Error(e)
					t.Error(e2)
				}
			}
			if found {
				return c
			}
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
		dev, _ := c.Devices.DeviceByName(cx)
		ms, err := checkMountpointUsage(dev.MountPoint)
		if err != nil {
			t.Error(err)
		}
		Log.WithFields(logrus.Fields{
			"name":       dev.Name,
			"mountPoint": dev.MountPoint,
			"size":       ms,
			"usedSize":   dev.UsedSize}).Info("Mountpoint usage info")
		if uint64(ms) != dev.UsedSize {
			t.Errorf("MountPoint: %q\n\t  Got Size: %d Expect: %d\n", dev.MountPoint, ms, dev.UsedSize)
		}
	}
	return c
}

func TestSyncNoCatalog(t *testing.T) {
	c := NewContext("../../testdata/filesync_freebooks")
	c.Files, _ = NewFileList(c)
	errs := Sync(c)
	expectErr := "No catalog data, nothing to do."
	var found bool
	for _, e := range errs {
		if e.Error() == expectErr {
			found = true
		}
	}
	if !found {
		t.Errorf("Expect: %q\n\t Got: %s", expectErr, spd.Sprint(errs))
	}
}

func TestSyncSimpleCopy(t *testing.T) {
	f := &syncTest{
		backupPath: "../../testdata/filesync_freebooks/",
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       28173338480,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0"),
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
	}
}

func TestSyncSimpleCopySourceFileError(t *testing.T) {
	testOutputDir := NewMountPoint(t, testTempDir, "mountpoint-0-")
	f := &syncTest{
		backupPath: "/root/",
		fileList: func() FileList {
			return FileList{
				File{
					Name:     "file",
					FileType: FILE,
					Size:     1024,
					Path:     "/root/file",
					DestPath: path.Join(testOutputDir, "file"),
					Mode:     0644,
					ModTime:  time.Now(),
					Owner:    os.Getuid(),
					Group:    os.Getgid(),
				},
			}
		},
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       42971520,
					MountPoint: testOutputDir,
				},
			}
		},
	}
	c := NewContext(f.backupPath)
	c.Files = f.fileList()
	c.Devices = f.deviceList()
	c.OutputStreamNum = f.outputStreams
	c.SplitMinSize = f.splitMinSize
	c.Catalog = NewCatalog(c)
	err2 := Sync(c)
	if len(err2) == 0 {
		t.Error("Expect: Errors  Got: No Errors")
	}
}

func TestSyncSimpleCopyDestPathError(t *testing.T) {
	testOutputDir := NewMountPoint(t, testTempDir, "mountpoint-0-")
	f := &syncTest{
		backupPath: fakeTestPath,
		fileList: func() FileList {
			return FileList{
				File{
					Name:     "testfile",
					FileType: FILE,
					Size:     41971520,
					Path:     path.Join(fakeTestPath, "testfile"),
					DestPath: path.Join(testOutputDir, "testfile"),
					Mode:     0444,
					ModTime:  time.Now(),
					Owner:    os.Getuid(),
					Group:    os.Getgid(),
				},
			}
		},
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       42971520,
					MountPoint: testOutputDir,
				},
			}
		},
	}
	c := NewContext(f.backupPath)
	c.Files = f.fileList()
	c.Devices = f.deviceList()
	c.OutputStreamNum = f.outputStreams
	c.SplitMinSize = f.splitMinSize
	c.Catalog = NewCatalog(c)

	// Create the destination files
	preSync(&(c.Devices)[0], &c.Catalog)

	// Make the file read only
	err := os.Chmod((c.Files)[0].DestPath, 0444)
	if err != nil {
		t.Error("Expect: Errors  Got: No Errors")
	}

	// Attempt to sync
	err2 := Sync(c)
	if len(err2) == 0 {
		t.Errorf("Expect: Errors  Got: Errors: %s", err2)
	}
}

func TestSyncPerms(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}
	testOutputDir := NewMountPoint(t, testTempDir, "mountpoint-0-")
	f := &syncTest{
		backupPath: fakeTestPath,
		fileList: func() FileList {
			return FileList{
				File{
					Name:     "diff_user",
					FileType: FILE,
					Size:     1024,
					Path:     path.Join(fakeTestPath, "diff_user"),
					DestPath: path.Join(testOutputDir, "diff_user"),
					Mode:     0640,
					ModTime:  time.Now(),
					Owner:    55000,
					Group:    55000,
				},
				File{
					Name:     "script.sh",
					FileType: FILE,
					Size:     1024,
					Path:     path.Join(fakeTestPath, "script.sh"),
					DestPath: path.Join(testOutputDir, "script.sh"),
					Mode:     0777,
					ModTime:  time.Now(),
					Owner:    os.Getuid(),
					Group:    os.Getgid(),
				},
				File{
					Name:     "some_dir",
					Path:     path.Join(fakeTestPath, "some_dir"),
					FileType: DIRECTORY,
					Size:     4096,
					DestPath: path.Join(testOutputDir, "some_dir"),
					Mode:     0755,
					ModTime:  time.Now(),
					Owner:    os.Getuid(),
					Group:    55000,
				},
			}
		},
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       28173338480,
					MountPoint: testOutputDir,
				},
			}
		},
		expectErrors: func() []error {
			return []error{
				SyncIncorrectOwnershipError{
					FilePath: filepath.Join(testOutputDir, "diff_user"),
					OwnerId:  55000,
					UserId:   os.Getuid(),
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
	}
}

func TestSyncSubDirs(t *testing.T) {
	f := &syncTest{
		backupPath: "../../testdata/filesync_directories/subdirs",
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       28173338480,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0"),
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
	}
}

func TestSyncSymlinks(t *testing.T) {
	f := &syncTest{
		backupPath: "../../testdata/filesync_symlinks/",
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       28173338480,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0"),
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
	}
}

func TestSyncBackupathIncluded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}
	f := &syncTest{
		backupPath: "../../testdata/filesync_freebooks",
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       28173338480,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0"),
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
	}
}

func TestSyncFileSplitAcrossDevices(t *testing.T) {
	f := &syncTest{
		backupPath:   "../../testdata/filesync_freebooks",
		splitMinSize: 1000,
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       1493583,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
				},
				Device{
					Name:       "Test Device 1",
					Size:       1000000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[1].Name,
			"mountPoint": c.Devices[1].MountPoint}).Print("Test mountpoint")
	}
}

func TestSyncAcrossDevicesNoSplit(t *testing.T) {
	f := &syncTest{
		backupPath:   "../../testdata/filesync_freebooks",
		splitMinSize: 1000,
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       668711 + 4096 + 4096,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
				},
				Device{
					Name:       "Test Device 1",
					Size:       1812584 + 4096 + 4096,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[1].Name,
			"mountPoint": c.Devices[1].MountPoint}).Print("Test mountpoint")
	}
}

func TestSyncFileSplitAcrossDevicesWithProgress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}
	f := &syncTest{
		splitMinSize: 1000,
		backupPath:   fakeTestPath,
		fileList: func() FileList {
			return FileList{
				File{
					Name:     "testfile",
					FileType: FILE,
					Size:     41971520,
					Path:     path.Join(fakeTestPath, "testfile"),
					Mode:     0644,
					ModTime:  time.Now(),
					Owner:    os.Getuid(),
					Group:    os.Getgid(),
				},
			}
		},
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       31485760,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
				},
				Device{
					Name:       "Test Device 1",
					Size:       10485760,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[1].Name,
			"mountPoint": c.Devices[1].MountPoint}).Print("Test mountpoint")
	}
}

func TestSyncLargeFileAcrossOneWholeDeviceAndHalfAnother(t *testing.T) {
	f := &syncTest{
		backupPath:   "../../testdata/filesync_large_binary_file/",
		splitMinSize: 1000,
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       9999999,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
				},
				Device{
					Name:       "Test Device 1",
					Size:       850000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
				},
			}
		},
		expectDeviceUsage: func() []expectDevice {
			return []expectDevice{
				expectDevice{
					name:      "Test Device 0",
					usedBytes: 9999999,
				},
				expectDevice{
					name:      "Test Device 1",
					usedBytes: 485760,
				},
			}
		},
	}
	c := runSyncTest(t, f)
	checkDevices(t, c, f.expectDeviceUsage())
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[1].Name,
			"mountPoint": c.Devices[1].MountPoint}).Print("Test mountpoint")
	}
}

func TestSyncLargeFileAcrossThreeDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}
	f := &syncTest{
		backupPath: "../../testdata/filesync_large_binary_file",
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       3499350,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
				},
				Device{
					Name:       "Test Device 1",
					Size:       3499350,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
				},
				Device{
					Name:       "Test Device 2",
					Size:       3499346,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-2-"),
				},
			}
		},
		expectDeviceUsage: func() []expectDevice {
			return []expectDevice{
				expectDevice{
					name:      "Test Device 0",
					usedBytes: 3499350,
				},
				expectDevice{
					name:      "Test Device 1",
					usedBytes: 3499350,
				},
				expectDevice{
					name:      "Test Device 2",
					usedBytes: 3499346,
				},
			}
		},
	}
	c := runSyncTest(t, f)
	checkDevices(t, c, f.expectDeviceUsage())
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[1].Name,
			"mountPoint": c.Devices[1].MountPoint}).Print("Test mountpoint")
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[2].Name,
			"mountPoint": c.Devices[2].MountPoint}).Print("Test mountpoint")
	}
}

// TestSyncDirsWithLotsOfFiles checks syncing directories with thousands of files and directories that _had_ thousands of
// files. Directories that had thousands of files are still allocated in the filesystem as containing thousands of file
// pointers, but since filesystems don't reclaim space, recreating these directories on the destination drive will allocate
// the blocksize of the device--4096 bytes.
func TestSyncDirsWithLotsOfFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}
	testTempDir, err := ioutil.TempDir(testTempDir, "gds-filetests-")
	if err != nil {
		t.Error(err)
	}
	// Copy filesync_test09_directories to the temp dir and delete all of the files in the dir
	cmd := exec.Command("/usr/bin/cp", "-R", "../../testdata/filesync_directories", testTempDir)
	err = cmd.Run()
	if err != nil {
		t.Error(err)
	}
	// Duplicate the sub dir
	cmd = exec.Command("/usr/bin/cp", "-R", path.Join(testTempDir, "filesync_directories", "dir_with_thousand_files"),
		path.Join(testTempDir, "filesync_directories", "dir_with_thousand_files_deleted"))
	err = cmd.Run()
	if err != nil {
		t.Error(err)
	}
	// Delete all of the files in the directory
	files, err := filepath.Glob(path.Join(testTempDir, "filesync_directories", "dir_with_thousand_files_deleted", "*"))
	if err != nil {
		t.Error(err)
	}
	for _, y := range files {
		err := os.Remove(y)
		if err != nil {
			t.Errorf("EXPECT: No errors GOT: Error: %s", err)
		}

	}

	f := &syncTest{
		backupPath: path.Join(testTempDir, "filesync_directories"),
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       4300000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
	}
}

func TestSyncLargeFileNotEnoughDeviceSpace(t *testing.T) {
	f := &syncTest{
		backupPath: "../../testdata/filesync_large_binary_file",
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       3499350,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
				},
				Device{
					Name:       "Test Device 1",
					Size:       3499350,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
				},
				Device{
					Name:       "Test Device 2",
					Size:       300000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-2-"),
				},
			}
		},
		expectErrors: func() []error {
			return []error{
				SyncNotEnoughDevicePoolSpace{
					backupSize:     10489856,
					devicePoolSize: 7298700,
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[1].Name,
			"mountPoint": c.Devices[1].MountPoint}).Print("Test mountpoint")
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[2].Name,
			"mountPoint": c.Devices[2].MountPoint}).Print("Test mountpoint")
	}
}

func TestSyncDestPathPermissionDenied(t *testing.T) {
	f := &syncTest{
		backupPath: "../../testdata/filesync_nowrite_perms/",
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       5675,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
	}
}

func TestSyncDestPathSha1sum(t *testing.T) {
	// hash for testdata/filesync_freebooks/alice/alice_in_wonderland_by_lewis_carroll_gutenberg.org.htm
	expectHash := "08cdd7178a20032c27d152a1f440334ee5f132a0"
	f := &syncTest{
		backupPath: "../../testdata/filesync_freebooks/alice/",
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					Size:       669000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0"),
				},
			}
		},
	}
	c := runSyncTest(t, f)
	fn, err := c.Files.FileByName("alice_in_wonderland_by_lewis_carroll_gutenberg.org.htm")
	if err != nil {
		t.Error(err)
	}
	hash, err := fn.DestPathSha1Sum()
	if err != nil {
		t.Error(err)
	}
	if c != nil {
		Log.WithFields(logrus.Fields{
			"srcHash":  expectHash,
			"destHash": hash}).Infoln("sha1sum of source and dest")
		if hash != expectHash {
			t.Error(err)
		}
		Log.WithFields(logrus.Fields{
			"deviceName": c.Devices[0].Name,
			"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
	}
}
