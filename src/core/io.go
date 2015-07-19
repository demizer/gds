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
	size              uint64
	totalBytesWritten uint64
	totalRead         uint64
	timeStart         time.Time
	progress          copyProgress
	sha1              hash.Hash
}

func NewIoReaderWriter(outFile io.Writer, outFileSize uint64) *IoReaderWriter {
	i := &IoReaderWriter{
		Writer:    outFile,
		size:      outFileSize,
		timeStart: time.Now(),
		sha1:      sha1.New(),
	}
	return i
}

func (i *IoReaderWriter) MultiWriter() io.Writer {
	return io.MultiWriter(i, i.sha1)
}

func (i *IoReaderWriter) Read(p []byte) (int, error) {
	n, err := i.Reader.Read(p)
	if err == nil {
		i.totalRead += uint64(n)
	}
	return n, err
}

type progressPoint struct {
	time              time.Time
	totalBytesWritten uint64
}

type copyProgress []progressPoint

func (c *copyProgress) addPoint(totalBytesWritten uint64) {
	*c = append(*c, progressPoint{
		time:              time.Now(),
		totalBytesWritten: totalBytesWritten,
	})
}

func (c *copyProgress) avgBytesPerSec(timeStart time.Time) uint64 {
	avgPoints := 10
	var a copyProgress
	// +1 to compensate for zero index
	if len(*c) < avgPoints+1 {
		a = (*c)[0:len(*c)]
		avgPoints = len(*c)
	} else {
		a = (*c)[len(*c)-(avgPoints+1) : len(*c)-1]
	}
	var sum uint64
	for _, y := range a {
		sum += y.totalBytesWritten
	}
	td := time.Since(timeStart).Seconds()
	if td == 0 {
		td = 1
	}
	return uint64(float64(sum/uint64(avgPoints)) / td)
}

func (c *copyProgress) lastPoint() progressPoint {
	if len(*c) == 0 {
		return (*c)[0]
	}
	return (*c)[len(*c)-1]
}

// Write writes to the io.Writer and also create a progress point for tracking
// write speed.
func (i *IoReaderWriter) Write(p []byte) (int, error) {
	n, err := i.Writer.Write(p)
	if err != nil {
		return n, err
	}
	i.totalBytesWritten += uint64(n)
	if len(i.progress) == 0 {
		i.progress.addPoint(i.totalBytesWritten)
		return n, err
	} else if (time.Since(i.progress.lastPoint().time).Seconds()) < 1 {
		return n, err
	}
	i.progress.addPoint(i.totalBytesWritten)
	return n, err
}

func (i *IoReaderWriter) WriteBytesPerSecond() uint64 {
	return i.progress.avgBytesPerSec(i.timeStart)
}

func (i *IoReaderWriter) Sha1SumToString() string {
	return hex.EncodeToString(i.sha1.Sum(nil))
}
