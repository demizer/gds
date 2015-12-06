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
	timeStart               time.Time   // The start time of the file copy
	timeLastReport          time.Time   // The time of the last report
	sizeTotal               uint64      // The total size of the input file
	sizeWritn               chan uint64 // Channel for reporting number of bytes written
	sizeWritnTotal          uint64      // Total number of bytes written to dest file
	sizeWritnFromLastReport uint64      // Number of bytes written to the dest file since last progress report
	sha1                    hash.Hash
}

func NewIoReaderWriter(outFile io.Writer, progressReport chan uint64, outFileSize uint64) *IoReaderWriter {
	i := &IoReaderWriter{
		Writer:    outFile,
		sizeTotal: outFileSize,
		timeStart: time.Now(),
		sizeWritn: progressReport,
		sha1:      sha1.New(),
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

		// Log.Debugf("i: %p i.sizeTotal: %d i.sizeWritnFromLastReport: %d n: %d time.Since: %f",
		// i, i.sizeTotal, i.sizeWritnFromLastReport, n, time.Since(i.timeLastReport).Seconds())

		// Limit the number of reports to once a second
		if i.timeLastReport.IsZero() || time.Since(i.timeLastReport).Seconds() > 1 || i.sizeWritnTotal == i.sizeTotal {
			Log.Debugf("REPORTING: %p, timeLastReport.IsZero: %t BYTES: %d FILE SIZE: %d",
				i, i.timeLastReport.IsZero(), i.sizeWritnFromLastReport, i.sizeTotal)
			i.sizeWritn <- i.sizeWritnFromLastReport
			Log.Debugln("REPORTING FINISHED")
			i.sizeWritnFromLastReport = 0
			i.timeLastReport = time.Now()
		}
	}
	return n, err
}

func (i *IoReaderWriter) Sha1SumToString() string {
	return hex.EncodeToString(i.sha1.Sum(nil))
}
