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

func sync(device *Device, files map[string][]*File, oio chan<- *syncerState, done chan<- bool) error {
	for _, f := range files[device.Name] {
		fName := filepath.Join(device.MountPoint, f.Name)
		// Open source file for reading
		sFile, err := os.Open(f.Path)
		defer sFile.Close()
		if err != nil {
			return fmt.Errorf("Could not open source file %q for writing! -- %s", f.Path, err)
		}
		var oFile io.Writer
		// Open dest file for writing
		if device.MountPoint == "/dev/null" {
			oFile = ioutil.Discard
		} else {
			oFile, err := os.Open(fName)
			defer oFile.Close()
			if err != nil {
				return fmt.Errorf("Could not open dest file %q for writing! -- %q", fName, err)
			}
		}
		mIo := NewIoReaderWriter(oFile, f.Size)
		fmt.Printf("Writing %q (%s)...\n", f.Name, humanize.IBytes(f.Size))
		nIo := mIo.MultiWriter()
		oio <- &syncerState{io: mIo, file: f}
		if oSize, err := io.Copy(nIo, sFile); err != nil {
			return fmt.Errorf("Error writing file %q, err: %q", fName, err)
		} else {
			device.UsedSize += uint64(oSize)
		}
		sFile.Close()
	}
	done <- true
	return nil
}

// Sync synchronizes files to mounted devices on mountpoints. Sync will copy
// new files, delete old files, and fix or update files on the destination
// device that do not match the source sha1 hash.
func Sync(c *Context) error {
	done := make(chan bool, 100)
	doneSp := make(chan bool, 100)
	ssc := make(chan *syncerState, 100)
	for i := 0; i < c.OutputStreamNum; i++ {
		go sync(&(c.Devices)[i], c.Catalog, ssc, done)
	}
	// for x := 0; x < c.OutputStreamNum; x++ {
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
		case sst := <-ssc:
			dspT += 1
			go sst.outputProgress(doneSp)
		default:
			if dc == c.OutputStreamNum && dsp == dspT {
				break loop
			}
		}
	}
	return nil
}
