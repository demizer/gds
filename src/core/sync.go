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
	fakeTestPath  string = "/fake/path/" // If the trailing slash is removed, tests will break.
	totalSyncSize uint64
)

// SyncProgress details information of the overall sync progress.
type SyncProgress struct {
	SizeWritn      uint64
	SizeTotal      uint64
	BytesPerSecond uint64
	ETA            time.Time
}

// SyncDeviceProgress details sync progress of an individual file. Channel is closed once writes to the device are complete.
type SyncDeviceProgress struct {
	FileName string
	FilePath string
	FileSize uint64

	FileSizeWritn      uint64 // Number of bytes written since last report
	FileTotalSizeWritn uint64 // Total number of bytes written to dest file
	FileBytesPerSecond uint64

	DeviceSizeWritn      uint64 // Number of bytes written since last report
	DeviceTotalSizeWritn uint64 // Total number of bytes written to dest device
	DeviceBytesPerSecond uint64
}

type tracker struct {
	io     *IoReaderWriter
	file   *File
	device *Device
	closed bool
}

func (s *tracker) report(dev *Device, devBps *bytesPerSecond, sfp chan<- SyncDeviceProgress) {
	// Tracks total file size reported to the tracker
	var size uint64
	// File bps calculation
	fbps := newBytesPerSecond()
	// Used to track times from the last report
	lastReport := time.Now()
outer:
	for {
		select {
		case bw := <-s.io.sizeWritn:
			Log.WithFields(logrus.Fields{
				"fileName":                   s.file.Name,
				"fileDestName":               s.file.DestPath,
				"fileBytesWritn":             bw,
				"fileTotalBytes":             s.file.DestSize,
				"elapsedTimeSinceLastReport": time.Since(lastReport),
				"copyTotalBytesWritn":        s.io.sizeWritnTotal,
			}).Infoln("Copy report")
			dev.SizeWritn += bw
			size += bw
			devBps.addPoint(bw)
			fbps.addPoint(size)
			sfp <- SyncDeviceProgress{
				FileName:             s.file.Name,
				FilePath:             s.file.DestPath,
				FileSize:             s.file.SourceSize,
				FileSizeWritn:        bw,
				FileTotalSizeWritn:   s.io.sizeWritnTotal,
				FileBytesPerSecond:   fbps.calc(),
				DeviceSizeWritn:      bw,
				DeviceTotalSizeWritn: dev.SizeWritn,
				DeviceBytesPerSecond: devBps.calc(),
			}
			if size == s.file.DestSize {
				Log.WithFields(logrus.Fields{
					"bw":                  bw,
					"destPath":            s.file.DestPath,
					"destSize":            s.file.DestSize,
					"s.io.sizeWritnTotal": s.io.sizeWritnTotal,
				}).Print("Copy complete")
				break outer
			}
			lastReport = time.Now()
			lastReport = time.Now()
		case <-time.After(time.Second):
			if s.closed {
				break outer
			}
			Log.Debugf("No bytes written to %q on device %q in last second.", s.file.Name, dev.Name)
			devBps.addPoint(0)
			fbps.addPoint(0)
			lastReport = time.Now()
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
	DeviceSizeWritn uint64
	DeviceSizeTotal uint64
	SyncContextSize uint64
}

func (e SyncNotEnoughDeviceSpaceForSyncContextError) Error() string {
	return fmt.Sprintf("Not enough space for sync context data file!"+
		"DeviceName=%q DeviceSizeWritn=%d DeviceSizeTotal=%d SyncContextSize=%d",
		e.DeviceName, e.DeviceSizeWritn, e.DeviceSizeTotal, e.SyncContextSize)
}

func saveSyncContext(c *Context, lastDevice *Device) (size int64, err error) {
	c.LastSyncEndDate = time.Now()
	sgzSize, err := syncContextCompressedSize(c)
	if err != nil {
		return
	}
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
func sync2dev(device *Device, catalog *Catalog, trakc chan<- tracker, cerr chan<- error) {
	Log.WithFields(logrus.Fields{"device": device.Name}).Infoln("Syncing to device")
	syncErrCtx := fmt.Sprintf("sync Device[%q]:", device.Name)
	for _, cf := range (*catalog)[device.Name] {
		if cf.err != nil {
			// An error was generated in pre-sync, send it down the line
			cerr <- cf.err
			continue
		}
		Log.WithFields(logrus.Fields{
			"fileName":       cf.Name,
			"device":         device.Name,
			"fileSourceSize": cf.SourceSize,
			"fileDestSize":   cf.DestSize,
			"fileSplitStart": cf.SplitStartByte,
			"fileSplitEnd":   cf.SplitEndByte}).Infoln("Syncing file")

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
		newTracker := tracker{io: mIo, file: cf, device: device}
		trakc <- newTracker
		Log.Debugln("TIME AFTER TRACKER SEND:", time.Since(ns))
		if cf.SplitEndByte == 0 && !syncTest {
			if _, err := io.Copy(nIo, sFile); err != nil {
				newTracker.closed = true
				Log.WithFields(logrus.Fields{
					"filePath":       cf.Path,
					"fileSourceSize": cf.SourceSize,
					"fileDestSize":   cf.DestSize,
					"deviceSize":     device.SizeTotal,
				}).Error("Error copying file!")
				cerr <- fmt.Errorf("%s copy %s: %s", syncErrCtx, cf.DestPath, err.Error())
				break
			} else {
				err = sFile.Close()
				err = oFile.Close()
				ls, err := os.Lstat(cf.DestPath)
				if err == nil {
					Log.WithFields(logrus.Fields{
						"file":     cf.Name,
						"size":     ls.Size(),
						"destSize": cf.DestSize,
					}).Debugln("File size")
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
			if oSize, err := io.CopyN(nIo, sFile, int64(cf.DestSize)); err != nil {
				newTracker.closed = true
				Log.WithFields(logrus.Fields{
					"oSize":         oSize,
					"cf.Path":       cf.Path,
					"cf.SourceSize": cf.SourceSize,
					"cf.DestSize":   cf.DestSize,
					"d.SizeTotal":   device.SizeTotal,
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
		}
		if err != nil {
			cerr <- fmt.Errorf("%s %s", syncErrCtx, err.Error())
			break
		}
	}
	Log.WithFields(logrus.Fields{"device": device.Name, "mountPoint": device.MountPoint}).Info("Sync to device complete")
	close(trakc)
	close(cerr)
}

// Sync synchronizes files to mounted devices on mountpoints. Sync will copy new files, delete old files, and fix or update
// files on the destination device that do not match the source sha1 hash. If disableContextSave is true, the context file
// will be NOT be dumped to the last devices as compressed JSON.
func Sync(c *Context, disableContextSave bool) []error {
	var retError []error
	var lastDevice *Device

	totalSyncSize = c.Catalog.TotalSize()
	Log.WithFields(logrus.Fields{"dataSize": totalSyncSize, "poolSize": c.Devices.TotalSize()}).Info("Data vs Pool size")

	i := 0
	streamCount := 0

	c.SyncStartDate = time.Now()

	progress := newBytesPerSecond()
	var progressLastSizeWritn uint64

	trackers := make(map[int]chan tracker) // Channel for communicating progress

	// Collects errors for final report
	errCollector := func(errs *[]error, eChan chan error) {
		select {
		case err, ok := <-eChan:
			if !ok {
				break
			}
			Log.Error(err)
			*errs = append(*errs, err)
		}
		Log.Debugln("Breaking error reporting loop!")
	}

	// Called every one second
	overallProgressReporter := func() {
		totalSyncBytesWritten := c.Devices.TotalSizeWritten()
		diffSizeWritn := totalSyncBytesWritten - progressLastSizeWritn
		Log.WithFields(logrus.Fields{
			"totalSyncBytesWritn": totalSyncBytesWritten,
			"lastBytesWritn":      progressLastSizeWritn,
			"diff":                diffSizeWritn,
		}).Infoln("Overall progress report")
		progress.addPoint(diffSizeWritn)
		c.SyncProgress <- SyncProgress{
			SizeWritn:      totalSyncBytesWritten,
			BytesPerSecond: progress.calc(),
		}
		progressLastSizeWritn = totalSyncBytesWritten
	}

	// Reports progress data to controlling goroutine
	progressReporter := func(dev *Device, trakIndex int, trakc <-chan tracker) {
		// device bps calculation
		dbps := newBytesPerSecond()
		for {
			// Report the progress for each file
			if progress, ok := <-trakc; ok {
				progress.report(dev, dbps, c.SyncDeviceProgress[trakIndex])
			} else {
				break
			}
		}
		close(c.SyncDeviceProgress[trakIndex])
		Log.WithFields(logrus.Fields{"trakIndex": trakIndex, "dev.SizeWritn": dev.SizeWritn}).Debugf("TRACKER DONE")
	}

	// GO GO GO
	for {
		if i == len(c.Catalog) && streamCount == 0 {
			Log.Debugln("Breaking main sync loop! Counter:", i)
			break
		}
		if i < len(c.Devices) && i < len(c.Catalog) && streamCount < c.OutputStreamNum {
			streamCount += 1
			// Launch into go routine in case exec is blocked waiting for a user to mount a device
			go func(index int) {
				Log.Debugln("Starting Sync() iteration", index)
				d := &(c.Devices)[index]

				// ENSURE DEVICE IS MOUNTED
				// Block until a reply is sent. Discard the value because it's not important.
				Log.Debugln("Sending SyncDeviceMount channel request to index", index)
				c.SyncDeviceMount[index] <- true
				<-c.SyncDeviceMount[index]
				Log.Debugf("Received response from SyncDeviceMount[%d] channel request", index)

				preSync(d, &c.Catalog)
				if len(c.Devices) > index {
					lastDevice = d
				}

				errorChan := make(chan error, 10)
				trackers[index] = make(chan tracker, 100)

				go errCollector(&retError, errorChan)

				go progressReporter(d, index, trackers[index])

				// Finally, starting syncing!
				go func(syncIndex int, dev *Device) {
					sync2dev(dev, &c.Catalog, trackers[syncIndex], errorChan)
					streamCount -= 1
					Log.Debugln("SYNC", syncIndex, "DONE")
				}(index, d)
			}(i)
			i += 1
		}
		overallProgressReporter()
		// Give exec some breathing room
		time.Sleep(time.Second)
	}

	// One final update to show full copy
	overallProgressReporter()

	if !disableContextSave {
		_, err := saveSyncContext(c, lastDevice)
		if err != nil {
			retError = append(retError, err)
		}
	}

	return retError
}
