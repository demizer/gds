package core

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

// HashFile is a file for which the hash will be computed. This is sent back on the report channel on each update.
type HashFile struct {
	FileName       string
	FilePath       string
	SizeWritn      uint64
	SizeWritnLast  uint64 // The number of bytes written since the last update
	SizeTotal      uint64
	BytesPerSecond BytesPerSecond
	file           *File
}

// HashComputer is the main hashing abstraction.
type HashComputer struct {
	Reports chan HashFile
	Files   []HashFile
	Errors  chan error
}

// NewSourceFileHashComputer returns a new hashing computer build from Files.
func NewSourceFileHashComputer(files FileIndex, errChan chan error) *HashComputer {
	var nFiles []HashFile
	for _, f := range files {
		if f.FileType == FILE && !strings.Contains(f.Path, fakeTestPath) {
			nFiles = append(nFiles, HashFile{
				FileName:  f.Name,
				FilePath:  f.Path,
				SizeTotal: f.Size,
				file:      f,
			})
		}
	}
	return &HashComputer{
		Reports: make(chan HashFile),
		Files:   nFiles,
		Errors:  errChan,
	}
}

func (h *HashComputer) report(wg *sync.WaitGroup, bw chan uint64, file HashFile) {
	defer wg.Done()
	for {
		if b, ok := <-bw; ok {
			file.SizeWritn += b
			file.SizeWritnLast = b
			file.BytesPerSecond.AddPoint(b)
			h.Reports <- file
			if file.SizeWritn == file.SizeTotal {
				break
			}
		}
	}
}

func (h *HashComputer) calc(f HashFile, done chan bool) {
	var sum string
	bw := make(chan uint64)
	var wg sync.WaitGroup

	wg.Add(1)
	go h.report(&wg, bw, f)

	Log.WithFields(logrus.Fields{"filePath": f.FilePath}).Infof("Computing sha1")
	tn := time.Now()
	hash := sha1.New()
	sio := NewIoReaderWriter(f.FilePath, hash, f.SizeTotal, bw, true, &done)

	file, err := os.Open(f.FilePath)
	if err != nil {
		h.Errors <- fmt.Errorf("calc: %s", err)
		goto end
	}
	defer file.Close()

	if _, err := io.Copy(sio, file); err != nil {
		h.Errors <- fmt.Errorf("calc: %s", err)
		goto end
	}

	sum = hex.EncodeToString(hash.Sum(nil))
	if err != nil {
		h.Errors <- err
		goto end
	}

	f.file.Sha1Sum = sum

end:
	wg.Wait()
	done <- true
	Log.WithFields(logrus.Fields{"hash": sum, "filePath": f.FilePath, "time": time.Since(tn)}).Infof("Hash calc finished")
	close(bw)
}

// ComputeAll will compute the hashes of all files. If a value is sent on the done channel, or it is closed, then the
// function exits.
func (h *HashComputer) ComputeAll(done chan bool) {
	runs := runtime.NumCPU()
	workDone := make(chan bool)
	var wg sync.WaitGroup
	x, count := 0, 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if count < runs && x < len(h.Files) {
				wg.Add(1)
				go h.calc(h.Files[x], workDone)
				count++
				x++
			}
			select {
			case <-done:
				return
			default:
				// runtime.Gosched() // This does not work so well here
				time.Sleep(time.Millisecond * 5) // Don't starve the main thread
			}
		}
	}()

	for {
		select {
		case <-workDone:
			wg.Done()
			count--
		case <-done:
			return
		}
	}

	wg.Wait()
	close(h.Reports)
}
