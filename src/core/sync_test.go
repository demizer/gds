package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
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

func syncTestChannelHandlers(c *Context) {
	for x := 0; x < len(c.Devices); x++ {
		c.SyncDeviceMount[x] = make(chan bool)
		c.SyncProgress[x] = make(chan SyncProgress, 10)
		c.SyncFileProgress[x] = make(chan SyncFileProgress, 10)
	}
	for x := 0; x < len(c.Devices); x++ {
		go func(index int) {
			for {
				select {
				case <-c.SyncDeviceMount[index]:
					c.SyncDeviceMount[index] <- true
				case <-c.SyncProgress[index]:
				case <-c.SyncFileProgress[index]:
				}
			}
		}(x)
	}
}

func runSyncTest(t *testing.T, f *syncTest) *Context {
	c := NewContext()
	c.BackupPath = f.backupPath
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
	if c.OutputStreamNum == 0 {
		c.OutputStreamNum = 1
	}
	c.SplitMinSize = f.splitMinSize
	c.Catalog, err = NewCatalog(c)
	if err != nil {
		t.Fatalf("EXPECT: No errors from NewCatalog() GOT: %s", err)
	}
	// spd.Dump(c)
	// os.Exit(1)

	// Mimic device mounting
	syncTestChannelHandlers(c)

	// Do the work!
	err2 := Sync(c, true)
	if len(err2) != 0 {
		found := false
		if f.expectErrors == nil {
			t.Errorf("Expect: No errors\n\t  Got: %+#v", err2)
			return nil
		}
		for _, e := range err2 {
			for _, e2 := range f.expectErrors() {
				if reflect.TypeOf(e) == reflect.TypeOf(e2) {
					found = true
				} else {
					t.Errorf("EXPECT: Error TypeOf %s GOT: Error TypeOf %s",
						reflect.TypeOf(e), reflect.TypeOf(e2))
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
			if cvf.SplitEndByte == 0 && uint64(ls.Size()) != cvf.SourceSize {
				t.Errorf("File: %q\n\t  Got Size: %d Expect: %d\n",
					cvf.DestPath, ls.Size, cvf.SourceSize)
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
		if uint64(ms) > dev.SizeTotal {
			t.Errorf("Mountpoint %q usage (%d bytes) is greater than device size (%d bytes)",
				dev.MountPoint, ms, dev.SizeTotal)
		}
		Log.WithFields(logrus.Fields{
			"1-name":        dev.Name,
			"2-mountPoint":  dev.MountPoint,
			"3-SizeOnDisk":  ms,
			"4-d.SizeWritn": dev.SizeWritn,
			"5-d.SizeTotal": dev.SizeTotal,
		}).Info("Mountpoint usage info")
		if uint64(ms) != dev.SizeWritn {
			t.Errorf("MountPoint: %q\n\t  Got Size: %d Expect: %d\n", dev.MountPoint, ms, dev.SizeTotal)
		}
	}
	return c
}

func TestSyncSimpleCopy(t *testing.T) {
	f := &syncTest{
		backupPath: "../../testdata/filesync_freebooks/",
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					SizeTotal:  28173338480,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0"),
					BlockSize:  4096,
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
					Name:       "file",
					FileType:   FILE,
					SourceSize: 1024,
					DestSize:   1024,
					Path:       "/root/file",
					DestPath:   path.Join(testOutputDir, "file"),
					Mode:       0644,
					ModTime:    time.Now(),
					Owner:      os.Getuid(),
					Group:      os.Getgid(),
				},
			}
		},
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					SizeTotal:  42971520,
					MountPoint: testOutputDir,
					BlockSize:  4096,
				},
			}
		},
	}
	var err error
	c := NewContext()
	c.BackupPath = f.backupPath
	c.Files = f.fileList()
	c.Devices = f.deviceList()
	c.SplitMinSize = f.splitMinSize
	c.Catalog, err = NewCatalog(c)
	if err != nil {
		t.Errorf("EXPECT: No errors from NewCatalog() GOT: %s", err)
	}
	// Mimic device mounting
	syncTestChannelHandlers(c)
	err2 := Sync(c, true)
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
					Name:       "testfile",
					FileType:   FILE,
					SourceSize: 41971520,
					Path:       path.Join(fakeTestPath, "testfile"),
					DestPath:   path.Join(testOutputDir, "testfile"),
					Mode:       0444,
					ModTime:    time.Now(),
					Owner:      os.Getuid(),
					Group:      os.Getgid(),
				},
			}
		},
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					SizeTotal:  42971520,
					MountPoint: testOutputDir,
					BlockSize:  4096,
				},
			}
		},
	}
	var err error
	c := NewContext()
	c.BackupPath = f.backupPath
	c.Files = f.fileList()
	c.Devices = f.deviceList()
	c.SplitMinSize = f.splitMinSize
	c.Catalog, err = NewCatalog(c)
	if err != nil {
		t.Errorf("EXPECT: No errors from NewCatalog() GOT: %s", err)
	}
	// Create the destination files
	preSync(&(c.Devices)[0], &c.Catalog)

	// Make the file read only
	err = os.Chmod((c.Files)[0].DestPath, 0444)
	if err != nil {
		t.Error("Expect: Errors  Got: No Errors")
	}

	// Mimic device mounting
	syncTestChannelHandlers(c)

	// Attempt to sync
	err2 := Sync(c, true)
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
					Name:       "diff_user",
					FileType:   FILE,
					SourceSize: 1024,
					Path:       path.Join(fakeTestPath, "diff_user"),
					DestPath:   path.Join(testOutputDir, "diff_user"),
					Mode:       0640,
					ModTime:    time.Now(),
					Owner:      55000,
					Group:      55000,
				},
				File{
					Name:       "script.sh",
					FileType:   FILE,
					SourceSize: 1024,
					Path:       path.Join(fakeTestPath, "script.sh"),
					DestPath:   path.Join(testOutputDir, "script.sh"),
					Mode:       0777,
					ModTime:    time.Now(),
					Owner:      os.Getuid(),
					Group:      os.Getgid(),
				},
				File{
					Name:       "some_dir",
					Path:       path.Join(fakeTestPath, "some_dir"),
					FileType:   DIRECTORY,
					SourceSize: 4096,
					DestPath:   path.Join(testOutputDir, "some_dir"),
					Mode:       0755,
					ModTime:    time.Now(),
					Owner:      os.Getuid(),
					Group:      55000,
				},
			}
		},
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					SizeTotal:  28173338480,
					MountPoint: testOutputDir,
					BlockSize:  4096,
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
					SizeTotal:  28173338480,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0"),
					BlockSize:  4096,
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
					SizeTotal:  28173338480,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0"),
					BlockSize:  4096,
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
					SizeTotal:  28173338480,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0"),
					BlockSize:  4096,
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
					SizeTotal:  1493583,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
					BlockSize:  4096,
				},
				Device{
					Name:       "Test Device 1",
					SizeTotal:  1010000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
					BlockSize:  4096,
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
					SizeTotal:  668711 + 4096 + 4096,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
					BlockSize:  4096,
				},
				Device{
					Name:       "Test Device 1",
					SizeTotal:  1812584 + 4096 + 4096 + 1000, // 1000 sync context data
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
					BlockSize:  4096,
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
					Name:       "testfile",
					FileType:   FILE,
					SourceSize: 41971520,
					Path:       path.Join(fakeTestPath, "testfile"),
					Mode:       0644,
					ModTime:    time.Now(),
					Owner:      os.Getuid(),
					Group:      os.Getgid(),
				},
			}
		},
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					SizeTotal:  31485760,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
					BlockSize:  4096,
				},
				Device{
					Name:       "Test Device 1",
					SizeTotal:  10495760,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
					BlockSize:  4096,
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
					SizeTotal:  9999999,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
					BlockSize:  4096,
				},
				Device{
					Name:       "Test Device 1",
					SizeTotal:  850000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
					BlockSize:  4096,
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
					usedBytes: 3500050,
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		checkDevices(t, c, f.expectDeviceUsage())
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
					SizeTotal:  3499350,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
					BlockSize:  4096,
				},
				Device{
					Name:       "Test Device 1",
					SizeTotal:  3499350,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
					BlockSize:  4096,
				},
				Device{
					Name:       "Test Device 2",
					SizeTotal:  3500346,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-2-"),
					BlockSize:  4096,
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
					usedBytes: 3500050,
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		checkDevices(t, c, f.expectDeviceUsage())
		// Log.WithFields(logrus.Fields{
		// "deviceName": c.Devices[0].Name,
		// "mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
		// Log.WithFields(logrus.Fields{
		// "deviceName": c.Devices[1].Name,
		// "mountPoint": c.Devices[1].MountPoint}).Print("Test mountpoint")
		// Log.WithFields(logrus.Fields{
		// "deviceName": c.Devices[2].Name,
		// "mountPoint": c.Devices[2].MountPoint}).Print("Test mountpoint")
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
					SizeTotal:  4300000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
					BlockSize:  4096,
				},
			}
		},
		expectDeviceUsage: func() []expectDevice {
			return []expectDevice{
				expectDevice{
					name:      "Test Device 0",
					usedBytes: 4182016,
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		checkDevices(t, c, f.expectDeviceUsage())
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
					SizeTotal:  3499350,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
					BlockSize:  4096,
				},
				Device{
					Name:       "Test Device 1",
					SizeTotal:  3499350,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
					BlockSize:  4096,
				},
				Device{
					Name:       "Test Device 2",
					SizeTotal:  300000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-2-"),
					BlockSize:  4096,
				},
			}
		},
		expectErrors: func() []error {
			return []error{CatalogNotEnoughDevicePoolSpaceError{}}
		},
	}
	c := NewContext()
	c.BackupPath = f.backupPath
	var err error
	c.Files, err = NewFileList(c)
	if err != nil {
		t.Errorf("EXPECT: No errors from NewFileList() GOT: %s", err)
	}
	c.Devices = f.deviceList()
	c.Catalog, err = NewCatalog(c)
	if err == nil || reflect.TypeOf(err) != reflect.TypeOf(f.expectErrors()[0]) {
		if err == nil {
			t.Error("EXPECT: Error TypeOf CatalogNotEnoughDevicePoolSpaceError GOT: nil")
		} else {
			t.Errorf("EXPECT: Error TypeOf CatalogNotEnoughDevicePoolSpaceError GOT: %T %q", err, err)
		}
	}
	// Log.Error(err)
	// Log.WithFields(logrus.Fields{
	// "deviceName": c.Devices[0].Name,
	// "mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
	// Log.WithFields(logrus.Fields{
	// "deviceName": c.Devices[1].Name,
	// "mountPoint": c.Devices[1].MountPoint}).Print("Test mountpoint")
}

func TestSyncDestPathPermissionDenied(t *testing.T) {
	f := &syncTest{
		backupPath: "../../testdata/filesync_nowrite_perms/",
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					SizeTotal:  10000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
					BlockSize:  4096,
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
					SizeTotal:  769000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0"),
					BlockSize:  4096,
				},
			}
		},
	}
	c := runSyncTest(t, f)
	if c != nil {
		fn, err := c.Files.FileByName("alice_in_wonderland_by_lewis_carroll_gutenberg.org.htm")
		if err != nil {
			t.Error(err)
		}
		hash, err := fn.DestPathSha1Sum()
		if err != nil {
			t.Error(err)
		}
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

func TestSyncSaveContextLastDevice(t *testing.T) {
	f := &syncTest{
		backupPath:   "../../testdata/filesync_freebooks",
		splitMinSize: 1000,
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					SizeTotal:  1493583,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
					BlockSize:  4096,
				},
				Device{
					Name:       "Test Device 1",
					SizeTotal:  1020000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
					BlockSize:  4096,
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

func TestSyncSaveContextLastDeviceNotEnoughSpaceError(t *testing.T) {
	f := &syncTest{
		backupPath:   "../../testdata/filesync_freebooks",
		splitMinSize: 1000,
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					SizeTotal:  1493583,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
					BlockSize:  4096,
				},
				Device{
					Name:       "Test Device 1",
					SizeTotal:  1016382,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
					BlockSize:  4096,
				},
			}
		},
		expectErrors: func() []error {
			return []error{
				SyncNotEnoughDeviceSpaceForSyncContextError{
					DeviceName:      "Test Device 1",
					DeviceUsed:      1000000,
					DeviceSize:      1000000,
					SyncContextSize: 890,
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

// TestSyncFromTempDirectory copies the testdata to the temp directory. This has the effect of reducing file sizes to their
// actual size. Once this is done, a sync is made to a real file system which creates small files using 1 block size.
func TestSyncFromTempDirectory(t *testing.T) {
	p, err := ioutil.TempDir("", "gds-freebooks-")
	if err != nil {
		t.Fatalf("EXPECT: path to temp mount GOT: %s", err)
	}
	Log.WithFields(logrus.Fields{"path": p}).Print("filesync_freebooks temporary path")
	cmd := exec.Command("/usr/bin/cp", "-R", "../../testdata/filesync_freebooks", p)
	err = cmd.Run()
	if err != nil {
		t.Fatalf("EXPECT: No copy errors GOT: %s", err)
	}
	f := &syncTest{
		backupPath:   p,
		splitMinSize: 1000,
		deviceList: func() DeviceList {
			return DeviceList{
				Device{
					Name:       "Test Device 0",
					SizeTotal:  1000000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
					BlockSize:  4096,
				},
				Device{
					Name:       "Test Device 1",
					SizeTotal:  1000000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
					BlockSize:  4096,
				},
				Device{
					Name:       "Test Device 2",
					SizeTotal:  1000000,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-2-"),
					BlockSize:  4096,
				},
			}
		},
	}
	runSyncTest(t, f)
}
