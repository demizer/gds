package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/demizer/go-humanize"
)

type syncerState struct {
	io   *IoReaderWriter
	file *File
}

func (s *syncerState) outputProgress(done chan<- bool) {
	var lp time.Time
	for {
		if lp.IsZero() {
			lp = time.Now()
		} else if time.Since(lp).Seconds() < 1 && s.io.totalBytesWritten != s.file.Size {
			continue
		} else if s.io.totalBytesWritten < s.file.Size || lp.IsZero() {
			fmt.Printf("\033[2KCopying %q (%s/%s) [%s/s]\n", s.file.Name,
				humanize.IBytes(s.io.totalBytesWritten),
				humanize.IBytes(s.file.Size),
				humanize.IBytes(s.io.WriteBytesPerSecond()))
			lp = time.Now()
		} else {
			fmt.Printf("Copy %q to %q completed [%s/s]\n", s.file.Name, s.file.DestPath,
				humanize.IBytes(s.io.WriteBytesPerSecond()))
			done <- true
			return
		}
	}
}

func setMetaData(f *File) error {
	err := os.Chown(f.DestPath, f.Owner, f.Group)
	if err != nil {
		return err
	}

	// Take a journey of a thousand steps to set atime and mtime on a symlink without following it...
	mTimeval := syscall.NsecToTimespec(f.ModTime.UnixNano())
	times := []syscall.Timespec{
		mTimeval,
		mTimeval,
	}
	err = LUtimesNano(f.DestPath, times)
	if err != nil {
		return err
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

func sync(device *Device, catalog *Catalog, oio chan<- *syncerState, done chan<- bool, cerr chan<- error) {
	reportErr := func(err error) {
		cerr <- err
		done <- true
	}
	for _, cf := range (*catalog)[device.Name] {
		if cf.Owner != os.Getuid() && os.Getuid() != 0 {
			cerr <- SyncIncorrectOwnershipError{cf.DestPath, cf.Owner, os.Getuid()}
			continue
		}
		// This is where directories are created and given the correct metadata.
		if cf.IsDir {
			err := os.Mkdir(cf.DestPath, cf.Mode)
			if err != nil {
				reportErr(err)
				return
			}
			err = setMetaData(cf)
			if err != nil {
				reportErr(err)
				return
			}
			continue
		}

		// Symlink handling
		if cf.IsSymlink {
			p, err := os.Readlink(cf.Path)
			if err != nil {
				reportErr(err)
				continue
			}
			err = os.Symlink(p, cf.DestPath)
			if err != nil {
				reportErr(err)
				continue
			}
			err = setMetaData(cf)
			if err != nil {
				reportErr(err)
				continue
			}
			continue
		}

		// Open source file for reading
		sFile, err := os.Open(cf.Path)
		defer sFile.Close()
		if err != nil {
			reportErr(err)
			return
		}
		var oFile *os.File
		// Open dest file for writing
		if device.MountPoint == "/dev/null" {
			oFile = ioutil.Discard.(*os.File)
		} else {
			oFile, err = os.Open(cf.DestPath)
			defer oFile.Close()
			if err != nil {
				oFile, err = os.Create(cf.DestPath)
				if err != nil {
					reportErr(err)
					return
				}
				err = os.Chmod(cf.DestPath, cf.Mode)
				if err != nil {
					reportErr(err)
					return
				}
			}
		}
		mIo := NewIoReaderWriter(oFile, cf.Size)
		nIo := mIo.MultiWriter()
		oio <- &syncerState{io: mIo, file: cf}
		if oSize, err := io.Copy(nIo, sFile); err != nil {
			reportErr(err)
			return
		} else {
			device.UsedSize += uint64(oSize)
		}
		cf.SrcSha1 = mIo.Sha1SumToString()
		err = setMetaData(cf)
		if err != nil {
			reportErr(err)
			return
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
	for i := 0; i < c.OutputStreamNum; i++ {
		go sync(&(c.Devices)[i], &c.Catalog, ssc, done, errorChan)
	}
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
	return retError
}
