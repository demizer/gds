package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

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
			fmt.Printf("Copy %q completed [%s/s]\n", s.file.Name,
				humanize.IBytes(s.io.WriteBytesPerSecond()))
			done <- true
			return
		}
	}
}

func sync(device *Device, catalog *Catalog, oio chan<- *syncerState, done chan<- bool, cerr chan<- error) {
	reportErr := func(err error) {
		cerr <- err
		done <- true
	}
	for _, cf := range (*catalog)[device.Name] {
		fName := filepath.Join(device.MountPoint, cf.Name)
		cf.DestPath = fName
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
			oFile, err = os.Open(fName)
			defer oFile.Close()
			if err != nil {
				oFile, err = os.Create(fName)
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
		sFile.Close()
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
