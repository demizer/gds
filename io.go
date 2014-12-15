package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"time"

	"github.com/dustin/go-humanize"
)

type IoReaderWriter struct {
	io.Reader
	io.Writer
	size         uint64
	totalWritten uint64
	totalRead    uint64
	timeStart    time.Time
	sha256       hash.Hash
}

func NewIoReaderWriter(outFile io.Writer, outFileSize uint64) *IoReaderWriter {
	i := &IoReaderWriter{
		Writer:    outFile,
		size:      outFileSize,
		timeStart: time.Now(),
		sha256:    sha256.New(),
	}
	return i
}

func (i *IoReaderWriter) MultiWriter() io.Writer {
	return io.MultiWriter(i, i.sha256)
}

func (i *IoReaderWriter) Read(p []byte) (int, error) {
	n, err := i.Reader.Read(p)
	if err == nil {
		i.totalRead += uint64(n)
		fmt.Println("Read", n, "bytes for a total of",
			humanize.IBytes(i.totalRead))
	}
	return n, err
}

func (i *IoReaderWriter) Write(p []byte) (int, error) {
	tNow := time.Since(i.timeStart)
	n, err := i.Writer.Write(p)
	if err == nil {
		i.totalWritten += uint64(n)
		var mbps uint64
		if uint64(tNow.Seconds()) > 0 {
			mbps = i.totalWritten / uint64(tNow.Seconds())
		}
		fmt.Printf("\rCopying [%s/%s] (%s/s)",
			humanize.IBytes(i.totalWritten),
			humanize.IBytes(i.size),
			humanize.Bytes(mbps),
		)
	}
	return n, err
}

func (i *IoReaderWriter) Sha256SumToString() string {
	return hex.EncodeToString(i.sha256.Sum(nil))
}
