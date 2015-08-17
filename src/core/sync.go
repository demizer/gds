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
	"sync"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/demizer/go-humanize"
)

// Used for testing
var fakeTestPath string = "/fake/path/" // If the trailing slash is removed, tests will break.

// SyncProgress details information of the overall sync progress.
type SyncProgress struct {
	DeviceName     string
	MountPoint     string
	SizeWritn      uint64
	SizeTotal      uint64
	BytesPerSecond uint64
	ETA            time.Time
}

// SyncFileProgress details sync progress of an individual file.
type SyncFileProgress struct {
	FileName       string
	FilePath       string
	SizeWritn      uint64
	SizeTotal      uint64
	BytesPerSecond uint64
}

type tracker struct {
	io     *IoReaderWriter
	file   *File
	device *Device
	closed bool
}

func (s *tracker) report(sp chan<- SyncProgress, sfp chan<- SyncFileProgress) {
	for {
		if s.closed {
			break
		}
		if s.file.SplitEndByte != 0 && s.io.totalBytesWritten < s.file.SplitEndByte-s.file.SplitStartByte {
			sfp <- SyncFileProgress{
				FileName:       s.file.Name,
				FilePath:       s.file.DestPath,
				SizeWritn:      s.io.totalBytesWritten,
				SizeTotal:      s.file.Size,
				BytesPerSecond: s.io.WriteBytesPerSecond(),
			}
		} else {
			Log.WithFields(logrus.Fields{
				"destPath":       s.file.DestPath,
				"bytesPerSecond": humanize.IBytes(s.io.WriteBytesPerSecond())}).Print("Copy complete")
			sp <- SyncProgress{
				DeviceName: s.device.Name,
				MountPoint: s.device.MountPoint,
				SizeWritn:  s.device.SizeWritn,
				SizeTotal:  s.device.SizeTotal,
			}
			break
		}
	}
}

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
	DeviceUsed      uint64
	DeviceSize      uint64
	SyncContextSize uint64
}

func (e SyncNotEnoughDeviceSpaceForSyncContextError) Error() string {
	return fmt.Sprintf("Not enough space for sync context data file! device=%q usedSize=%d size=%d syncDataSize=%d",
		e.DeviceName, e.DeviceUsed, e.DeviceSize, e.SyncContextSize)
}

func saveSyncContext(c *Context, lastDevice *Device) (size int64, err error) {
	c.LastSyncEndDate = time.Now()
	sgzSize, err := syncContextCompressedSize(c)
	if err != nil {
		return
	}
	if uint64(sgzSize)+lastDevice.SizeWritn > lastDevice.SizeWritn {
		err = SyncNotEnoughDeviceSpaceForSyncContextError{
			lastDevice.Name, lastDevice.SizeWritn, lastDevice.SizeWritn, uint64(sgzSize),
		}
		return
	}
	cp := filepath.Join(lastDevice.MountPoint, "sync_context_"+c.LastSyncStartDate.Format(time.RFC3339)+".json.gz")
	f, err := os.Create(cp)
	err = writeCompressedContextToFile(c, f)
	if err != nil {
		return
	}
	s, err := os.Lstat(cp)
	if err == nil {
		size = s.Size()
		Log.WithFields(logrus.Fields{
			"sync_context_file": cp,
			"size":              size,
		}).Debug("Saved sync context on last device")
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
		// fmt.Println(xy.DestPath)
		createFile(xy)
	}
}

// sync2dev is the main file syncing function. It is big, mean, and will eat your bytes.
func sync2dev(device *Device, catalog *Catalog, trakc chan<- tracker, cerr chan<- error) {
	Log.WithFields(logrus.Fields{"device": device.Name}).Infoln("Syncing to device")
	for _, cf := range (*catalog)[device.Name] {
		if cf.err != nil {
			// An error was generated in pre-sync, send it down the line
			cerr <- cf.err
			continue
		}
		Log.WithFields(logrus.Fields{
			"file_name":        cf.Name,
			"device":           device.Name,
			"file_size":        cf.Size,
			"file_split_start": cf.SplitStartByte,
			"file_split_end":   cf.SplitEndByte}).Infoln("Syncing file")

		// We only need to check the size of the file if it is a directory or a symlink
		if cf.FileType == DIRECTORY || cf.FileType == SYMLINK {
			ls, err := os.Lstat(cf.DestPath)
			if err != nil {
				cerr <- fmt.Errorf("sync Lstat: %s", err.Error())
				continue
			}
			Log.WithFields(logrus.Fields{"file": cf.DestPath, "size": ls.Size()}).Debugln("File size")
			device.SizeWritn += uint64(ls.Size())
			continue
		}

		var oFile *os.File
		var err error
		// Open dest file for writing
		oFile, err = os.OpenFile(cf.DestPath, os.O_RDWR, cf.Mode)
		if err != nil {
			cerr <- fmt.Errorf("sync ofile open: %s", err.Error())
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
			cerr <- fmt.Errorf("sync sfile open: %s", err.Error())
			continue
		}

		// Seek to the correct position for split files
		if cf.SplitStartByte != 0 && cf.SplitEndByte != 0 {
			_, err = sFile.Seek(int64(cf.SplitStartByte), 0)
			if err != nil {
				cerr <- fmt.Errorf("sync seek: %s", err.Error())
				continue
			}
		}

		mIo := NewIoReaderWriter(oFile, cf.Size)
		nIo := mIo.MultiWriter()

		ns := time.Now()
		newTracker := tracker{io: mIo, file: cf, device: device}
		trakc <- newTracker
		Log.Debugln("TIME AFTER TRACKER SEND:", time.Since(ns))
		if cf.SplitEndByte == 0 && !syncTest {
			if _, err := io.Copy(nIo, sFile); err != nil {
				// code smell ...
				newTracker.closed = true
				Log.WithFields(logrus.Fields{
					"filePath":    cf.Path,
					"fileSize":    cf.Size,
					"deviceUsage": device.SizeWritn,
					"deviceSize":  device.SizeTotal,
				}).Error("Error copying file!")
				cerr <- fmt.Errorf("sync copy %s: %s", cf.DestPath, err.Error())
				break
			} else {
				err = sFile.Close()
				err = oFile.Close()
				ls, err := os.Lstat(cf.DestPath)
				if err == nil {
					Log.WithFields(logrus.Fields{"file": cf.DestPath, "size": ls.Size()}).Debugln("File size")
					device.SizeWritn += uint64(ls.Size())
					// Set mode after file is copied to prevent no write perms from causing trouble
					err = os.Chmod(cf.DestPath, cf.Mode)
					if err == nil {
						Log.WithFields(logrus.Fields{
							"file": cf.Name,
							"mode": cf.Mode,
						}).Debugln("Set mode")
					}
				}
			}
		} else {
			var cb uint64
			if syncTest && cf.SplitEndByte == 0 {
				cb = cf.Size
			} else {
				cb = cf.SplitEndByte - cf.SplitStartByte
				if cf.SplitStartByte == 0 {
					cb = cf.SplitEndByte
				}
			}
			if oSize, err := io.CopyN(nIo, sFile, int64(cb)); err != nil {
				// code smell ...
				newTracker.closed = true
				cerr <- fmt.Errorf("sync copyn: %s", err.Error())
				break
			} else {
				Log.WithFields(logrus.Fields{"file": cf.DestPath, "size": oSize}).Debugln("File size")
				err = sFile.Close()
				err = oFile.Close()
				var ls os.FileInfo
				ls, err = os.Lstat(cf.DestPath)
				if err == nil {
					Log.WithFields(logrus.Fields{"file": cf.DestPath, "size": ls.Size()}).Debugln("File size")
					device.SizeWritn += uint64(ls.Size())
				}
			}
		}
		if err == nil {
			cf.SrcSha1 = mIo.Sha1SumToString()
			Log.WithFields(logrus.Fields{"file": cf.DestPath, "sha1sum": cf.SrcSha1}).Debugln("File sha1sum")
			err = setMetaData(cf)
		}
		if err != nil {
			cerr <- fmt.Errorf("sync: %s", err.Error())
			break
		}
	}
	// deviceSyncDone <- true
	Log.WithFields(logrus.Fields{
		"device":     device.Name,
		"mountPoint": device.MountPoint,
	}).Info("Sync to device complete")
	close(trakc)
}

// Sync synchronizes files to mounted devices on mountpoints. Sync will copy new files, delete old files, and fix or update
// files on the destination device that do not match the source sha1 hash. If disableContextSave is true, the context file
// will be NOT be dumped to the last devices as compressed JSON.
func Sync(c *Context, disableContextSave bool) []error {
	var retError []error
	var lastDevice *Device
	var wg sync.WaitGroup

	Log.WithFields(logrus.Fields{
		"dataSize": c.Files.TotalDataSize(),
		"poolSize": c.Devices.DevicePoolSize(),
	}).Info("Data vs Pool size")

	c.LastSyncStartDate = time.Now()

	wgFin := make(chan bool)
	errorChan := make(chan error, 100)
	go func() {
		for {
			select {
			case err := <-errorChan:
				retError = append(retError, err)
			case <-wgFin:
				Log.Debugln("Breaking error reporting loop!")
				return
			}
		}
	}()

	i := 0
	for i < len(c.Catalog) {
		Log.Debugln("Starting Sync() iteration", i)
		trackers := make(map[int]chan tracker) // Channel for communicating progress trackers
		for j := 0; j < c.OutputStreamNum; j++ {
			d := &(c.Devices)[i+j]
			// ENSURE DEVICE IS MOUNTED
			// Block until a reply is sent. Discard the value because it's not important.
			Log.Debugln("Sending SyncDeviceMount channel request to index", i+j)
			c.SyncDeviceMount[i+j] <- true
			<-c.SyncDeviceMount[i+j]
			Log.Debugln("Received response from SyncDeviceMount channel request")
			preSync(d, &c.Catalog)
			if len(c.Devices) > i+j {
				lastDevice = d
			}
			trackers[i+j] = make(chan tracker, 100)
			wg.Add(1)
			go func(index int, dev *Device) {
				defer func() {
					wg.Done()
					Log.Debugln("SYNC DEFER WG.DONE()")
				}()
				sync2dev(dev, &c.Catalog, trackers[index], errorChan)
			}(i+j, d)
		}
		for x, y := range trackers {
			wg.Add(1)
			f := c.Catalog[(c.Devices)[x].Name]
			go func(index int, trakc <-chan tracker, fileNum int) {
				defer func() {
					wg.Done()
					Log.Debugln("TRACKER", index, "DEFER WG.DONE()")
				}()
				for {
					ns := time.Now()
					progress, ok := <-trakc
					if !ok {
						break
					}
					Log.Debugln("TIME AFTER TRACKER RECEIVE:", time.Since(ns))
					ns = time.Now()
					progress.report(c.SyncProgress[index], c.SyncFileProgress[index])
					Log.Debugln("TIME AFTER TRACKER REPORT:", time.Since(ns))
				}
			}(x, y, len(f))
		}
		Log.Debugln("Waiting for goroutines to end in main loop...")
		wg.Wait()
		Log.Debugln("Sync() iteration", i, "finished!")
		i++
	}
	if !disableContextSave {
		s, err := saveSyncContext(c, lastDevice)
		if err != nil {
			retError = append(retError, err)
		} else {
			lastDevice.SizeWritn += uint64(s)
		}
	}
	close(wgFin)
	return retError
}
