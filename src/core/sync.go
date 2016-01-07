package core

import (
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demizer/go-humanize"
)

var (
	// Used for testing
	fakeTestPath string = "/fake/path/" // If the trailing slash is removed, tests will break.
)

func writeCompressedContextToFile(c *Context, f *os.File) (err error) {
	var jc []byte
	if err == nil {
		jc, err = json.Marshal(c)
	}
	gz, err := gzip.NewWriterLevel(f, flate.BestCompression)
	defer gz.Close()
	_, err = gz.Write(jc)
	return
}

// syncContextCompressedSize returns the actual file size of the sync context compressed into a file.
func syncContextCompressedSize(c *Context) (size int64, err error) {
	c.LastSyncEndDate = time.Now()
	f, err := ioutil.TempFile("", "gds-sync-context-")
	defer f.Close()
	if err != nil {
		return
	}
	err = writeCompressedContextToFile(c, f)
	if err != nil {
		return
	}
	s, err := os.Lstat(f.Name())
	if err == nil {
		size = s.Size()
	}
	return
}

type SyncNotEnoughDeviceSpaceForSyncContextError struct {
	DeviceName      string
	DeviceSizeWritn uint64
	DeviceSizeTotal uint64
	SyncContextSize uint64
}

func (e SyncNotEnoughDeviceSpaceForSyncContextError) Error() string {
	return fmt.Sprintf("Not enough space for sync context data file!"+
		"DeviceName=%q DeviceSizeWritn=%d DeviceSizeTotal=%d SyncContextSize=%d",
		e.DeviceName, e.DeviceSizeWritn, e.DeviceSizeTotal, e.SyncContextSize)
}

func saveSyncContext(c *Context) (size int64, err error) {
	c.LastSyncEndDate = time.Now()
	sgzSize, err := syncContextCompressedSize(c)
	if err != nil {
		return
	}
	lastDevice := &(c.Devices)[len(c.Devices)-1]
	if uint64(sgzSize)+lastDevice.SizeWritn > lastDevice.SizeTotal {
		err = SyncNotEnoughDeviceSpaceForSyncContextError{
			lastDevice.Name, lastDevice.SizeWritn, lastDevice.SizeTotal, uint64(sgzSize),
		}
		return
	}
	cp := filepath.Join(lastDevice.MountPoint, "sync_context_"+c.SyncStartDate.Format(time.RFC3339)+".json.gz")
	f, err := os.Create(cp)
	err = writeCompressedContextToFile(c, f)
	if err != nil {
		return
	}
	s, err := os.Lstat(cp)
	if err == nil {
		size = s.Size()
		Log.WithFields(logrus.Fields{"syncContextFile": cp, "size": size}).Debug("Saved sync context on last device")
	}
	return
}

func setMetaData(f *File) error {
	var err error
	mTimeval := syscall.NsecToTimespec(f.ModTime.UnixNano())
	times := []syscall.Timespec{
		mTimeval,
		mTimeval,
	}
	err = os.Chown(f.DestPath, f.Owner, f.Group)
	if err == nil {
		Log.WithFields(logrus.Fields{"owner": f.Owner, "group": f.Group}).Debugln("Set owner")
		// Change the modtime of a symlink without following it
		err = LUtimesNano(f.DestPath, times)
		if err == nil {
			Log.WithFields(logrus.Fields{"modTime": f.ModTime}).Debugln("Set modification time")
		}
	}
	if err != nil {
		return fmt.Errorf("setMetaData: %s", err.Error())
	}
	return nil
}

// SyncIncorrectOwnershipError is an error given when a file is encountered that is not owned by the current user. This error
// does not occur if the current user is root.
type SyncIncorrectOwnershipError struct {
	FilePath string
	OwnerId  int
	UserId   int
}

// Error implements the Error interface.
func (e SyncIncorrectOwnershipError) Error() string {
	return fmt.Sprintf("Cannot copy %q (owner id: %d) as user id: %d", e.FilePath, e.OwnerId, e.UserId)
}

// SyncNotEnoughDevicePoolSpaceError is an error given when the backup size exceeds the device pool storage size.
type SyncNotEnoughDevicePoolSpaceError struct {
	BackupSize     uint64
	DevicePoolSize uint64
}

// Error implements the Error interface.
func (e SyncNotEnoughDevicePoolSpaceError) Error() string {
	return fmt.Sprintf("Not enough device pool space! backup_size: %d (%s) device_pool_size: %d (%s)", e.BackupSize,
		humanize.IBytes(e.BackupSize), e.DevicePoolSize, humanize.IBytes(e.DevicePoolSize))
}

// createFile is a helper function for creating directories, symlinks, and regular files. If it encounters errors creating
// these files, the error is sent on the cerr buffered error channel.
func createFile(f *File) {
	var err error
	if f.Owner != os.Getuid() && os.Getuid() != 0 {
		f.err = SyncIncorrectOwnershipError{f.DestPath, f.Owner, os.Getuid()}
		Log.Errorf("createFile: %s", f.err)
		return
	}
	switch f.FileType {
	case FILE:
		var oFile *os.File
		if _, lerr := os.Stat(f.DestPath); lerr != nil {
			oFile, err = os.Create(f.DestPath)
			err = oFile.Close()
			if err == nil {
				Log.WithFields(logrus.Fields{"path": f.DestPath}).Debugln("Created file")
			}
		}
	case DIRECTORY:
		if _, lerr := os.Stat(f.DestPath); lerr != nil {
			err = os.Mkdir(f.DestPath, f.Mode)
			if err == nil {
				Log.WithFields(logrus.Fields{"path": f.DestPath}).Debugln("Created directory")
			}
		}
	case SYMLINK:
		if _, lerr := os.Lstat(f.DestPath); lerr != nil {
			p, _ := os.Readlink(f.Path)
			err = os.Symlink(p, f.DestPath)
			if err == nil {
				Log.WithFields(logrus.Fields{"path": f.DestPath}).Debugln("Created symlink")
			}
		}
	}
	if err == nil {
		err = setMetaData(f)
		if err != nil {
			f.err = fmt.Errorf("createFile: %s", err.Error())
		}
	}
}

// preSync uses the catalog to pre-create the files that are to be synced at the mountpoint. This allows for increased
// accuracy when calculating actual device usage during the sync. This is needed because directory files on Linux increase in
// size once they contain pointers to files and sub-directories.
func preSync(d *Device, c *Catalog) {
	for _, xy := range (*c)[d.Name] {
		createFile(xy)
	}
}

// sync2dev is the main file syncing function. It is big, mean, and will eat your bytes.
func sync2dev(device *Device, catalog *Catalog, trakc chan<- fileTracker, cerr chan<- error) {
	Log.WithFields(logrus.Fields{"device": device.Name}).Infoln("Syncing to device")
	syncErrCtx := fmt.Sprintf("sync Device[%q]:", device.Name)
	for _, cf := range (*catalog)[device.Name] {
		if cf.err != nil {
			// An error was generated in pre-sync, send it down the line
			cerr <- cf.err
			continue
		}
		Log.WithFields(logrus.Fields{"fileName": cf.Name, "device": device.Name,
			"fileSourceSize": cf.SourceSize, "fileDestSize": cf.DestSize,
			"fileSplitStart": cf.SplitStartByte, "fileSplitEnd": cf.SplitEndByte}).Infoln("Syncing file")

		var oFile *os.File
		var err error
		// Open dest file for writing
		oFile, err = os.OpenFile(cf.DestPath, os.O_RDWR, cf.Mode)
		if err != nil {
			cerr <- fmt.Errorf("%s ofile open: %s", syncErrCtx, err.Error())
			continue
		}
		defer oFile.Close()

		var sFile *os.File
		var syncTest bool
		if strings.Contains(cf.Path, fakeTestPath) {
			// For testing
			syncTest = true
			sFile, err = os.Open("/dev/urandom")
		} else {
			sFile, err = os.Open(cf.Path)
			defer sFile.Close()
		}
		if err != nil {
			cerr <- fmt.Errorf("%s sfile open: %s", syncErrCtx, err.Error())
			continue
		}

		// Seek to the correct position for split files
		if cf.SplitStartByte != 0 && cf.SplitEndByte != 0 {
			_, err = sFile.Seek(int64(cf.SplitStartByte), 0)
			if err != nil {
				cerr <- fmt.Errorf("%s seek: %s", syncErrCtx, err.Error())
				continue
			}
		}

		pReporter := make(chan uint64, 100)
		mIo := NewIoReaderWriter(oFile, pReporter, cf.DestSize)
		nIo := mIo.MultiWriter()

		ns := time.Now()
		ft := fileTracker{io: mIo, file: cf, device: device}
		select {
		case trakc <- ft:
			Log.Debugln("TIME AFTER FILE TRACKER SEND:", time.Since(ns))
		case <-time.After(200 * time.Second):
			panic("Should not be here! No receive on tracker channel in 200 seconds...")
		}
		if cf.SplitEndByte == 0 && !syncTest {
			if _, err := io.Copy(nIo, sFile); err != nil {
				ft.closed = true
				Log.WithFields(logrus.Fields{"filePath": cf.Path, "fileSourceSize": cf.SourceSize,
					"fileDestSize": cf.DestSize, "deviceSize": device.SizeTotal,
				}).Error("Error copying file!")
				cerr <- fmt.Errorf("%s copy %s: %s", syncErrCtx, cf.DestPath, err.Error())
				break
			} else {
				err = sFile.Close()
				err = oFile.Close()
				ls, err := os.Lstat(cf.DestPath)
				if err == nil {
					Log.WithFields(logrus.Fields{
						"file": cf.Name, "size": ls.Size(), "destSize": cf.DestSize,
					}).Debugln("File size")
					// Set mode after file is copied to prevent no write perms from causing trouble
					err = os.Chmod(cf.DestPath, cf.Mode)
					if err == nil {
						Log.WithFields(logrus.Fields{"file": cf.Name,
							"mode": cf.Mode}).Debugln("Set mode")
					}
				}
			}
		} else {
			if oSize, err := io.CopyN(nIo, sFile, int64(cf.DestSize)); err != nil {
				ft.closed = true
				Log.WithFields(logrus.Fields{
					"oSize": oSize, "cf.Path": cf.Path,
					"cf.SourceSize": cf.SourceSize, "cf.DestSize": cf.DestSize,
					"d.SizeTotal": device.SizeTotal,
				}).Error("Error copying file!")
				cerr <- fmt.Errorf("%s copyn: %s", syncErrCtx, err.Error())
				break
			} else {
				err = sFile.Close()
				err = oFile.Close()
			}
		}
		if err == nil {
			cf.SrcSha1 = mIo.Sha1SumToString()
			Log.WithFields(logrus.Fields{"file": cf.DestPath, "sha1sum": cf.SrcSha1}).Infoln("File sha1sum")
			err = setMetaData(cf)
			// For zero length files, report zero on the sizeWritn channel. io.Copy will only
			// create the file, but it will not report bytes written since there are none.
			// Otherwise sends to the tracker will block causing everything to grind to a halt.
			if cf.SourceSize == 0 && cf.FileType == FILE {
				mIo.sizeWritn <- 0
			}
		} else {
			cerr <- fmt.Errorf("%s %s", syncErrCtx, err.Error())
			break
		}
	}
	Log.WithFields(logrus.Fields{"device": device.Name, "mountPoint": device.MountPoint}).Info("Sync to device complete")
	close(trakc)
	// close(cerr)
}

func syncLaunch(c *Context, index int, err chan error, done chan bool) {
	Log.Debugln("Starting Sync() iteration", index)
	d := &(c.Devices)[index]

	// ENSURE DEVICE IS MOUNTED
	// Block until a reply is sent. Discard the value because it's not important.
	Log.Debugln("Sending SyncDeviceMount channel request to index", index)
	c.SyncDeviceMount[index] <- true
	<-c.SyncDeviceMount[index]
	Log.Debugf("Received response from SyncDeviceMount[%d] channel request", index)

	// Creates the destination files
	preSync(d, &c.Catalog)

	go c.SyncProgress.deviceCopyReporter(index)

	// Finally, starting syncing!
	sync2dev(d, &c.Catalog, c.SyncProgress.Device[index].files, err)

	done <- true
	Log.Debugln("SYNC", index, "DONE")
}

// Sync synchronizes files to mounted devices on mountpoints. Sync will copy new files, delete old files, and fix or update
// files on the destination device that do not match the source sha1 hash. If disableContextSave is true, the context file
// will be NOT be dumped to the last devices as compressed JSON.
func Sync(c *Context, disableContextSave bool, errChan chan error) {
	Log.WithFields(logrus.Fields{
		"dataSize": c.Catalog.TotalSize(), "poolSize": c.Devices.TotalSize(),
	}).Info("Data vs Pool size")

	// GO GO GO
	i, streamCount := 0, 0
	c.SyncStartDate = time.Now()
	done := make(chan bool, len(c.Devices))

	for {
		if i == len(c.Catalog) && streamCount == 0 {
			Log.Debugln("Breaking main sync loop! Counter:", i)
			break
		}
		if i < len(c.Devices) && i < len(c.Catalog) && streamCount < c.OutputStreamNum {
			streamCount += 1
			// Launch into go routine in case exec is blocked waiting for a user to mount a device
			go syncLaunch(c, i, errChan, done)
			i += 1
		} else {
			select {
			case <-done:
				streamCount -= 1
			case <-time.After(time.Second):
				c.SyncProgress.report()
			}
		}
	}

	// One final update to show full copy
	c.SyncProgress.report()

	if !disableContextSave {
		_, err := saveSyncContext(c)
		if err != nil {
			errChan <- err
		}
	}
}
