package core

import (
	"crypto/sha1"
	"encoding/hex"
	"hash"
	"io"
	"time"
)

type IoReaderWriter struct {
	io.Reader
	io.Writer
	destPath                string
	timeStart               time.Time   // The start time of the file copy
	timeLastReport          time.Time   // The time of the last report
	sizeTotal               uint64      // The total size of the input file
	sizeWritn               chan uint64 // Channel for reporting number of bytes written
	sizeWritnTotal          uint64      // Total number of bytes written to dest file
	sizeWritnFromLastReport uint64      // Number of bytes written to the dest file since last progress report
	done                    *chan bool  // If closed, copy will exit with DoneSignalReceived

	sha1 hash.Hash
}

// NewIoReaderWriter takes an output file, a channel of uint64 for reporting bytes written, and total number of bytes
// written. If ns is true, then sha1 hash computation is not used.
func NewIoReaderWriter(dp string, of io.Writer, ofs uint64, pr chan uint64, ns bool, done *chan bool) *IoReaderWriter {
	i := &IoReaderWriter{
		destPath:  dp,
		Writer:    of,
		sizeTotal: ofs,
		timeStart: time.Now(),
		sizeWritn: pr,
		done:      done,
	}
	if !ns {
		i.sha1 = sha1.New()
	}
	return i
}

func (i *IoReaderWriter) MultiWriter() io.Writer {
	return io.MultiWriter(i, i.sha1)
}

// Write writes to the io.Writer and also create a progress point for tracking write speed.
func (i *IoReaderWriter) Write(p []byte) (int, error) {
	n, err := i.Writer.Write(p)
	if err == nil {
		i.sizeWritnFromLastReport += uint64(n)
		i.sizeWritnTotal += uint64(n)

		// Log.Debugf("File Size: %d i: %p i.sizeTotal: %d i.sizeWritnFromLastReport: %d n: %d time.Since: %f",
		// i.sizeTotal, i, i.sizeTotal, i.sizeWritnFromLastReport, n, time.Since(i.timeLastReport).Seconds())

		ns := time.Now()
		report := func() {
			i.sizeWritn <- i.sizeWritnFromLastReport
			Log.Debugf("REPORTING FINISHED (%q) in %s FILE SIZE: %d", i.destPath, time.Since(ns), i.sizeTotal)
			if i.sizeWritnTotal != i.sizeTotal {
				i.sizeWritnFromLastReport = 0
				i.timeLastReport = time.Now()
			}
		}

		// Limit the number of reports to once a second
		if i.sizeWritnTotal == i.sizeTotal {
			Log.Debugf("REPORTING: %q (%p) -- FILE WRITE COMPLETE -- timeSinceLastReport %s BYTES: %d FILE SIZE: %d",
				i.destPath, i, time.Since(i.timeLastReport), i.sizeWritnFromLastReport, i.sizeTotal)
			report()
		} else if i.timeLastReport.IsZero() {
			Log.Debugf("REPORTING: %q (%p) timeLastReport.IsZero: %t BYTES: %d FILE SIZE: %d",
				i.destPath, i, i.timeLastReport.IsZero(), i.sizeWritnFromLastReport, i.sizeTotal)
			report()
		} else if time.Since(i.timeLastReport).Seconds() > 1 {
			Log.Debugf("REPORTING: %q (%p) timeSinceLastReport %s BYTES: %d FILE SIZE: %d",
				i.destPath, i, time.Since(i.timeLastReport), i.sizeWritnFromLastReport, i.sizeTotal)
			report()
		}
	}
	select {
	case _, ok := <-*i.done:
		if !ok {
			return n, new(DoneSignalReceived)
		}
	default:
	}
	return n, err
}

func (i *IoReaderWriter) Sha1SumToString() string {
	return hex.EncodeToString(i.sha1.Sum(nil))
}
