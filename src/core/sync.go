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
	"time"

	"github.com/Sirupsen/logrus"
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
func syncContextCompressedSize(c *Context) (size uint64, err error) {
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
		size = uint64(s.Size())
	}
	return
}

type SyncNotEnoughDeviceSpaceForSyncContextError struct {
	DeviceName            string
	DeviceSizeWritn       uint64
	DeviceSizeTotalPadded uint64
	SyncContextSize       uint64
}

func (e SyncNotEnoughDeviceSpaceForSyncContextError) Error() string {
	return fmt.Sprintf("Not enough space for sync context data file! "+
		"DeviceName=%q DeviceSizeWritn=%d DeviceSizeTotalPadded=%d SyncContextSize=%d",
		e.DeviceName, e.DeviceSizeWritn, e.DeviceSizeTotalPadded, e.SyncContextSize)
}

func saveSyncContext(c *Context) (size uint64, err error) {
	c.LastSyncEndDate = time.Now()
	sgzSize, err := syncContextCompressedSize(c)
	if err != nil {
		return
	}
	lastDevice := c.Devices[len(c.Devices)-1]
	if uint64(sgzSize)+lastDevice.SizeWritn > lastDevice.SizeTotalPadded() {
		err = SyncNotEnoughDeviceSpaceForSyncContextError{
			lastDevice.Name, lastDevice.SizeWritn, lastDevice.SizeTotalPadded(), uint64(sgzSize),
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
		size = uint64(s.Size())
		Log.WithFields(logrus.Fields{"syncContextFile": cp, "size": size}).Debug("Saved sync context on last device")
	}
	return
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

// SyncDestinatonFileOpenError is generated if an error occurrs when trying to open the destination file for writing.
type SyncDestinatonFileOpenError struct {
	err error
}

// Error implements the Error interface.
func (e SyncDestinatonFileOpenError) Error() string {
	return e.err.Error()
}

// SyncSourceFileOpenError is generated if an error occurrs when trying to open the destination file for writing.
type SyncSourceFileOpenError struct {
	err error
}

// Error implements the Error interface.
func (e SyncSourceFileOpenError) Error() string {
	return e.err.Error()
}

// sync2dev is the main file syncing function. It is big, mean, and will eat your bytes.
func sync2dev(device *Device, files *FileIndex, trakc chan<- fileTracker, cerr chan<- error) {
	Log.WithFields(logrus.Fields{"device": device.Name}).Infoln("Syncing to device")

	syncErrCtx := fmt.Sprintf("sync Device[%q]:", device.Name)

	for _, d := range files.DeviceFiles(device) {

		d.df.createFile(d.f)
		if d.df.err != nil {
			cerr <- d.df.err
			continue
		}

		if d.df.err != nil {
			// An error was generated in pre-sync, send it down the line
			cerr <- d.df.err
			continue
		}
		Log.WithFields(logrus.Fields{"fileName": d.f.Name, "device": device.Name,
			"fileSourceSize": d.f.Size, "fileDestSize": d.df.Size,
			"fileSplitStart": d.df.StartByte, "fileSplitEnd": d.df.EndByte}).Infoln("Syncing file")

		var oFile *os.File
		var err error
		// Open dest file for writing
		oFile, err = os.OpenFile(d.df.Path, os.O_RDWR, d.f.Mode)
		if err != nil {
			cerr <- SyncDestinatonFileOpenError{fmt.Errorf("%s ofile open: %s", syncErrCtx, err.Error())}
			continue
		}
		defer oFile.Close()

		var sFile *os.File
		var syncTest bool
		if strings.Contains(d.f.Path, fakeTestPath) {
			// For testing
			syncTest = true
			sFile, err = os.Open("/dev/urandom")
		} else {
			sFile, err = os.Open(d.f.Path)
			defer sFile.Close()
		}
		if err != nil {
			cerr <- SyncSourceFileOpenError{fmt.Errorf("%s sfile open: %s", syncErrCtx, err.Error())}
			continue
		}

		// Seek to the correct position for split files
		if d.f.IsSplit() {
			_, err = sFile.Seek(int64(d.df.StartByte), 0)
			if err != nil {
				cerr <- fmt.Errorf("%s seek: %s", syncErrCtx, err.Error())
				continue
			}
		}

		pReporter := make(chan uint64, 100)
		mIo := NewIoReaderWriter(oFile, pReporter, d.df.Size)
		nIo := mIo.MultiWriter()

		ns := time.Now()
		ft := fileTracker{io: mIo, f: d.f, df: d.df, device: device, done: make(chan bool)}
		select {
		case trakc <- ft:
			Log.Debugln("TIME AFTER FILE TRACKER SEND:", time.Since(ns))
		case <-time.After(200 * time.Second):
			panic("Should not be here! No receive on tracker channel in 200 seconds...")
		}
		if !d.f.IsSplit() && !syncTest {
			if _, err := io.Copy(nIo, sFile); err != nil {
				ft.closed = true
				Log.WithFields(logrus.Fields{"filePath": d.df.Path, "fileSourceSize": d.f.Size,
					"fileDestSize": d.df.Size, "deviceSize": device.SizeTotal,
				}).Error("Error copying file!")
				cerr <- fmt.Errorf("%s copy %s: %s", syncErrCtx, d.df.Path, err.Error())
				break
			} else {
				err = sFile.Close()
				err = oFile.Close()
				ls, err := os.Lstat(d.df.Path)
				if err == nil {
					Log.WithFields(logrus.Fields{
						"file": d.f.Name, "size": ls.Size(), "destSize": d.df.Size,
					}).Debugln("File size")
					// Set mode after file is copied to prevent no write perms from causing trouble
					err = os.Chmod(d.df.Path, d.f.Mode)
					if err == nil {
						Log.WithFields(logrus.Fields{"file": d.f.Name,
							"mode": d.f.Mode}).Debugln("Set mode")
					}
				}
			}
		} else {
			if oSize, err := io.CopyN(nIo, sFile, int64(d.df.Size)); err != nil {
				ft.closed = true
				Log.WithFields(logrus.Fields{
					"oSize": oSize, "d.df.Path": d.df.Path,
					"file.Size": d.f.Size, "d.df.Size": d.df.Size,
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
			d.f.Sha1Sum = mIo.Sha1SumToString()
			Log.WithFields(logrus.Fields{"file": d.df.Path, "sha1sum": d.f.Sha1Sum}).Infoln("File sha1sum")
			err = d.df.setMetaData(d.f)
			// For zero length files, report zero on the sizeWritn channel. io.Copy will only
			// create the file, but it will not report bytes written since there are none.
			// Otherwise sends to the tracker will block causing everything to grind to a halt.
			if d.f.Size == 0 && d.f.FileType == FILE {
				mIo.sizeWritn <- 0
			}
		} else {
			cerr <- fmt.Errorf("%s %s", syncErrCtx, err.Error())
			break
		}
		// Wait for the filetracker reporter to complete
		<-ft.done
	}
	Log.WithFields(logrus.Fields{"device": device.Name, "mountPoint": device.MountPoint}).Info("Sync to device complete")
}

func syncLaunch(c *Context, index int, err chan error, done chan bool) {
	Log.Debugln("Starting Sync() iteration", index)
	d := c.Devices[index]

	// ENSURE DEVICE IS MOUNTED
	// Block until a reply is sent. Discard the value because it's not important.
	Log.Debugln("Sending SyncDeviceMount channel request to index", index)
	c.SyncDeviceMount[index] <- true
	<-c.SyncDeviceMount[index]
	Log.Debugf("Received response from SyncDeviceMount[%d] channel request", index)

	go c.SyncProgress.deviceCopyReporter(index)

	// Finally, starting syncing!
	sync2dev(d, &c.FileIndex, c.SyncProgress.Device[index].files, err)

	done <- true

	close(c.SyncProgress.Device[index].Report)
	close(c.SyncProgress.Device[index].files)

	Log.Debugln("SYNC", index, "DONE")
}

// Sync synchronizes files to mounted devices on mountpoints. Sync will copy new files, delete old files, and fix or update
// files on the destination device that do not match the source sha1 hash. If disableContextSave is true, the context file
// will be NOT be dumped to the last devices as compressed JSON.
func Sync(c *Context, disableContextSave bool, errChan chan error) {
	Log.WithFields(logrus.Fields{
		"dataSize": c.FileIndex.TotalSize(), "poolSizePadded": c.Devices.TotalSizePadded(),
	}).Info("Data vs Pool size")

	// GO GO GO
	var streamCount uint16
	i := 0
	c.SyncStartDate = time.Now()
	done := make(chan bool, len(c.Devices))

	for {
		if i == c.DevicesUsed && streamCount == 0 {
			Log.Debugln("Breaking main sync loop! Counter:", i)
			break
		}
		if i < len(c.Devices) && i < c.DevicesUsed && streamCount < c.OutputStreamNum {
			streamCount += 1
			// Launch into go routine in case exec is blocked waiting for a user to mount a device
			go syncLaunch(c, i, errChan, done)
			i += 1
		} else {
			select {
			case <-done:
				streamCount -= 1
			case <-time.After(time.Second):
				c.SyncProgress.report(false)
			}
		}
	}

	// One final update to show full copy
	c.SyncProgress.report(true)

	close(c.SyncProgress.Report)

	if !disableContextSave {
		var err error
		c.SyncContextSize, err = saveSyncContext(c)
		if err != nil {
			errChan <- err
		}
	}
}
