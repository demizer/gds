package core

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"time"

	"github.com/demizer/go-humanize"

	log "gopkg.in/inconshreveable/log15.v2"
)

type IoReaderWriter struct {
	io.Reader
	io.Writer
	size              uint64
	totalBytesWritten uint64
	totalRead         uint64
	timeStart         time.Time
	sha1              hash.Hash
	lastProgress      time.Time
	lastBytesWritten  uint64
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
		log.Info("Read stats", "bytes", n, "total_bytes",
			humanize.IBytes(i.totalRead))
	}
	return n, err
}

type progressPoint struct {
	time                time.Time
	durationFromStart   time.Duration
	bytesWrittenInPoint uint64
	totalBytes          uint64
	totalBytesWritten   uint64
}

type copyProgress []progressPoint

func (c *copyProgress) averageBPS() uint64 {
	avgPoints := 20
	var a copyProgress
	// +1 to compensate for zero index
	if len(*c) < avgPoints+1 {
		a = (*c)[0:len(*c)]
	} else {
		a = (*c)[len(*c)-(avgPoints+1) : len(*c)-1]
	}
	var sum uint64
	// var l uint64
	// var ld time.Duration
	for _, y := range a {
		// td := uint64((y.durationFromStart - ld) / time.Second)
		// td := uint64(y.durationFromStart - ld)
		// log.Info("durationFromLast", "x", td)
		// fmt.Println((y.durationFromStart - ld) * time.Second)
		// fmt.Println((y.totalBytesWritten - l), uint64((y.durationFromStart - ld)))

		// if td == 0 {
		// td = 1
		// }
		// TODO: Is this right??? -- * 1000 to change to second from ms
		// sum += (y.totalBytes - y.totalBytesWritten - l) /// td //(td * 1000 * 1000)
		sum += y.bytesWrittenInPoint /// td //(td * 1000 * 1000)
		// l = y.totalBytesWritten
		// ld = y.durationFromStart
	}
	// fmt.Println(sum / uint64(avgPoints))
	return sum / uint64(avgPoints)
}

var progress copyProgress

func (i *IoReaderWriter) Write(p []byte) (int, error) {
	n, err := i.Writer.Write(p)
	if err != nil {
		return n, err
	}
	i.totalBytesWritten += uint64(n)

	if i.lastProgress.IsZero() {
		i.lastProgress = time.Now()
		// fmt.Println("IN HERE YO")
		return n, err
	} else if (time.Since(i.lastProgress).Seconds()) < 1 {
		// fmt.Println(time.Since(i.lastProgress) * time.Second)
		// fmt.Println(time.Since(i.lastProgress).Seconds())
		// i.lastWrite = time.Now()
		return n, err
	}

	// Display progress output
	// lc := progress.LastPoint()
	a := progressPoint{
		time:                time.Now(),
		durationFromStart:   time.Since(i.timeStart),
		bytesWrittenInPoint: i.totalBytesWritten - i.lastBytesWritten,
	}
	progress = append(progress, a)
	// a.durationDiff = lc.time.Add(a.durationInMs)
	// a.writeDiff =
	// fmt.Printf("Copying [%s/%s] (%s/s)\n",
	fmt.Printf("\rCopying [%s/%s] (%s/s)        ",
		humanize.IBytes(i.totalBytesWritten),
		humanize.IBytes(i.size),
		humanize.IBytes(progress.averageBPS()),
	)
	i.lastProgress = time.Now()
	i.lastBytesWritten = i.totalBytesWritten
	return n, err
}

func (i *IoReaderWriter) Sha1SumToString() string {
	return hex.EncodeToString(i.sha1.Sum(nil))
}
