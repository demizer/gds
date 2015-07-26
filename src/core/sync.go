package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	"github.com/demizer/go-humanize"

	log "gopkg.in/inconshreveable/log15.v2"
)

type syncerState struct {
	io     *IoReaderWriter
	file   *File
	closed bool
}

func (s *syncerState) outputProgress(done chan<- bool) {
	var lp time.Time
	showProgress := func() {
		fmt.Printf("\033[2KCopying %q (%s/%s) [%s/s]\n", s.file.Name,
			humanize.IBytes(s.io.totalBytesWritten),
			humanize.IBytes(s.file.Size),
			humanize.IBytes(s.io.WriteBytesPerSecond()))
	}
	for {
		if s.closed {
			done <- true
			return
		}
		if lp.IsZero() {
			lp = time.Now()
		} else if time.Since(lp).Seconds() < 1 &&
			((s.io.totalBytesWritten != s.file.Size) ||
				(s.io.totalBytesWritten != s.file.SplitEndByte-s.file.SplitStartByte)) {
			continue
		} else if lp.IsZero() || s.file.SplitEndByte == 0 && s.io.totalBytesWritten < s.file.Size {
			showProgress()
			lp = time.Now()
		} else if s.file.SplitEndByte != 0 && s.io.totalBytesWritten < s.file.SplitEndByte-s.file.SplitStartByte {
			showProgress()
			lp = time.Now()
		} else {
			fmt.Printf("Copy %q completed [%s/s]\n", s.file.Name, humanize.IBytes(s.io.WriteBytesPerSecond()))
			done <- true
			return
		}
	}
}

func setMetaData(f *File, cErr chan<- error) error {
	var err error
	if f.FileType != SYMLINK {
		err = os.Chmod(f.DestPath, f.Mode)
		if err != nil {
			cErr <- fmt.Errorf("setMetaData chmod: %s", err.Error())
		} else {
			log.Debug("Set mode", "file", f.DestPath, "mode", f.Mode)
		}
	}
	err = os.Chown(f.DestPath, f.Owner, f.Group)
	if err != nil {
		cErr <- fmt.Errorf("setMetaData chown: %s", err.Error())
	} else {
		log.Debug("Set owner", "owner", f.Owner, "group", f.Group)
	}
	// Change the modtime of a symlink without following it
	mTimeval := syscall.NsecToTimespec(f.ModTime.UnixNano())
	times := []syscall.Timespec{
		mTimeval,
		mTimeval,
	}
	err = LUtimesNano(f.DestPath, times)
	if err != nil {
		cErr <- fmt.Errorf("setMetaData LUtimesNano: %s", err.Error())
	}
	return nil
}

type SyncIncorrectOwnershipError struct {
	FilePath string
	OwnerId  int
	UserId   int
}

func (e SyncIncorrectOwnershipError) Error() string {
	return fmt.Sprintf("Cannot copy %q (owner id: %d) as user id: %d", e.FilePath, e.OwnerId, e.UserId)
}

func createFile(f *File, cerr chan<- error) {
	var err error
	switch f.FileType {
	case FILE:
		var oFile *os.File
		if _, lerr := os.Stat(f.DestPath); lerr != nil {
			oFile, err = os.Create(f.DestPath)
			log.Debug("Created file", "path", f.DestPath)
			oFile.Close()
		}
	case DIRECTORY:
		err = os.Mkdir(f.DestPath, f.Mode)
		log.Debug("Created Directory", "path", f.DestPath)
	case SYMLINK:
		p, err := os.Readlink(f.Path)
		if err != nil {
			break
		}
		err = os.Symlink(p, f.DestPath)
		log.Debug("Created Symlink", "path", f.DestPath)
	}
	if err != nil {
		cerr <- fmt.Errorf("createFile: %s", err.Error())
	} else {
		err = setMetaData(f, cerr)
		if err != nil {
			cerr <- fmt.Errorf("createFile: %s", err.Error())
		}
	}
}

// preSync uses the catalog to pre-create the files that are to be synced at the mountpoint. This allows for increased
// accuracy when calculating actual device usage during the sync. This is needed because directory files on Linux increase in
// size once they contain pointers to files and sub-directories.
func preSync(d *Device, c *Catalog) {
	// Discard any errors, they will be caught by the main sync process
	dErr := make(chan<- error, 100)
	for _, xy := range (*c)[d.Name] {
		createFile(xy, dErr)
	}
}

func sync(device *Device, catalog *Catalog, oio chan<- *syncerState, done chan<- bool, cerr chan<- error) {
	log.Info("Syncing to device", "device", device.Name)
	preSync(device, catalog)
	for _, cf := range (*catalog)[device.Name] {
		if cf.Owner != os.Getuid() && os.Getuid() != 0 {
			cerr <- SyncIncorrectOwnershipError{cf.DestPath, cf.Owner, os.Getuid()}
			continue
		}

		// We only need to check the size of the file if it is a directory or a symlink
		if cf.FileType == DIRECTORY || cf.FileType == SYMLINK {
			ls, _ := os.Lstat(cf.DestPath)
			log.Debug("File size", "file", cf.DestPath, "size", ls.Size())
			device.UsedSize += uint64(ls.Size())
			continue
		}

		var oFile *os.File
		// Open dest file for writing
		if device.MountPoint == "/dev/null" {
			oFile = ioutil.Discard.(*os.File)
		} else {
			oFile, _ = os.OpenFile(cf.DestPath, os.O_RDWR, cf.Mode)
		}
		defer oFile.Close()

		// Open source file for reading
		if cf.Path == "/dev/zero" {
			cf.SplitStartByte = 0
			cf.SplitEndByte = 4096
		}
		sFile, err := os.Open(cf.Path)
		defer sFile.Close()
		if err != nil {
			cerr <- fmt.Errorf("sync sfile open: %s", err.Error())
			break
		}

		// Seek to the correct position for split files
		if cf.SplitStartByte != 0 && cf.SplitEndByte != 0 {
			sFile.Seek(int64(cf.SplitStartByte), 0)
		}

		mIo := NewIoReaderWriter(oFile, cf.Size)
		nIo := mIo.MultiWriter()
		sst := &syncerState{io: mIo, file: cf}
		oio <- sst
		if cf.SplitEndByte == 0 {
			if oSize, err := io.Copy(nIo, sFile); err != nil {
				// code smell ...
				sst.closed = true
				cerr <- fmt.Errorf("sync copy: %s", err.Error())
				break
			} else {
				log.Debug("File size", "file", cf.DestPath, "size", oSize)
				device.UsedSize += uint64(oSize)
				sFile.Close()
				oFile.Close()
			}
		} else {
			cb := cf.SplitEndByte - cf.SplitStartByte
			if cf.SplitStartByte == 0 {
				cb = cf.SplitEndByte
			}
			if oSize, err := io.CopyN(nIo, sFile, int64(cb)); err != nil {
				// code smell ...
				sst.closed = true
				cerr <- fmt.Errorf("sync copyn: %s", err.Error())
				break
			} else {
				log.Debug("File size", "file", cf.DestPath, "size", oSize)
				device.UsedSize += uint64(oSize)
				sFile.Close()
				oFile.Close()
			}
		}

		cf.SrcSha1 = mIo.Sha1SumToString()
		log.Debug("File sha1sum", "file", cf.DestPath, "sha1sum", cf.SrcSha1)

		err = setMetaData(cf, cerr)
		if err != nil {
			cerr <- fmt.Errorf("sync: %s", err.Error())
			break
		}
	}
	done <- true
}

// Sync synchronizes files to mounted devices on mountpoints. Sync will copy
// new files, delete old files, and fix or update files on the destination
// device that do not match the source sha1 hash.
func Sync(c *Context) []error {
	done := make(chan bool, 100)
	doneSp := make(chan bool, 100)
	errorChan := make(chan error, 100)
	var retError []error
	ssc := make(chan *syncerState, 100)
	if c.OutputStreamNum == 0 {
		c.OutputStreamNum = 1
	}
	i := 0
	for i < len(c.Catalog) {
		for j := 0; j < c.OutputStreamNum; j++ {
			go sync(&(c.Devices)[i+j], &c.Catalog, ssc, done, errorChan)
		}
		i += c.OutputStreamNum
		// TODO: CODE SMELL... need a better concurrency pattern here.
		// Look at http://tip.golang.org/pkg/sync/#WaitGroup
		dc := 0
		dsp := 0
		dspT := 0
	loop:
		for {
			select {
			case <-done:
				dc += 1
			case <-doneSp:
				dsp += 1
			case err := <-errorChan:
				retError = append(retError, err)
			case sst := <-ssc:
				dspT += 1
				go sst.outputProgress(doneSp)
			default:
				// fmt.Println(dc, c.OutputStreamNum, dsp, dspT)
				if dc == c.OutputStreamNum && dsp == dspT {
					break loop
				}
			}
		}
	}
	return retError
}
