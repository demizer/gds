package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/demizer/go-elog"
	"github.com/dustin/go-humanize"
)

var spd = spew.ConfigState{Indent: "\t"} //, DisableMethods: true}

const BACKUP_PATH = "/mnt/data"
const DEST_PATH = "/mnt/backup"

const PADDING = 10485760 // 10G

type Drive struct {
	Name      string
	UsedSize  uint64 // Size in bytes
	TotalSize uint64
}

var myDrives = []Drive{
	{
		Name:      "WD SE 4TB 01",
		TotalSize: 3650140728 * 1024,
	},
	{
		Name:      "Hitachi 2TB 01",
		TotalSize: 1824966852 * 1024,
	},
	{
		Name:      "Hitachi 2TB 02",
		TotalSize: 1824966852 * 1024,
	},
}

type File struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Path    string      `json:"path"`
	Size    uint64      `json:"size"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"modTime"`
	Owner   uint32      `json:"owner"`
	Group   uint32      `json:"group"`
	SrcSha  string      `json:"srcSha"`
	DestSha string      `json:"destSha"`
}

var FILES []File

type BadFileMetadatError struct {
	Info      *File
	JsonError error
}

func (b *BadFileMetadatError) Error() string {
	return fmt.Sprintf("%s\n\n%s\n", b.JsonError, spd.Sdump(b.Info))
}

type BadFileMetadatErrors []error

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

func calcTotalBackupSpace() uint64 {
	var total uint64
	for _, x := range myDrives {
		total += x.TotalSize
	}
	return total
}

func calcTotalDataSize() uint64 {
	var totalDataSize uint64
	for _, x := range FILES {
		totalDataSize += x.Size
	}
	return totalDataSize
}

func verifyDestinationFileSHA(file File) (string, error) {
	dFile, err := os.Open(file.Path)
	defer dFile.Close()
	if err != nil {
		return "", err
	}
	s := sha256.New()
	if dSize, err := io.Copy(s, dFile); err != nil {
		return "", err
	} else if dSize != int64(file.Size) {
		return "", fmt.Errorf("A problem occurred reading %q! "+
			"Source Size (%d) != Destination Size (%d)",
			file.Path, file.Size, dSize)
	}
	dSha := hex.EncodeToString(s.Sum(nil))
	if file.SrcSha != dSha {
		return "", fmt.Errorf("Source SHA (%s) != Dest SHA (%s)",
			file.SrcSha, dSha)
	}
	return dSha, nil
}

func checkDestDriveSize(drive int, x File) int {
	d := myDrives[drive]
	if d.UsedSize+x.Size+PADDING >= d.TotalSize {
		var s string
		log.Printf("Drive %q is full! (%s of %s) Mount new drive "+
			"and press enter to continue...",
			d.Name, humanize.IBytes(d.UsedSize),
			humanize.IBytes(d.UsedSize))
		fmt.Scanf("%s", &s)
		return drive + 1
	}
	return drive
}

func WriteStuff() error {
	mdName := filepath.Join(os.Getenv("HOME"),
		time.Now().Format(time.RFC3339)+"_bigdata_files.json")
	log.Infof("Logging to %q\n", mdName)
	mOut, err := os.Create(mdName)
	if err != nil {
		return err
	}
	defer mOut.Close()
	var drive int
	writeJson := func(f *File) error {
		var v []byte
		v, jErr := json.Marshal(f)
		if jErr != nil {
			// Sometimes the mod time is not formatted correctly
			f.ModTime = time.Now()
			var err2 error
			v, err2 = json.Marshal(f)
			if err2 != nil {
				// bail
				return &BadFileMetadatError{f, err2}
			}
		}
		mdBuf := bytes.NewBuffer(v)
		_, err = io.Copy(mOut, mdBuf)
		if err != nil {
			return err
		}
		return nil
	}
	for _, x := range FILES {
		drive = checkDestDriveSize(drive, x)

		sFile, err := os.Open(x.Path)
		if err != nil {
			return err
		}
		defer sFile.Close()

		oFile, err := os.Create(filepath.Join(DEST_PATH, x.ID))
		if err != nil {
			return err
		}
		defer oFile.Close()

		mIo := NewIoReaderWriter(oFile, x.Size)
		log.Printf("Writing %q (%s)...\n\r",
			x.Name, humanize.IBytes(x.Size))
		nIo := mIo.MultiWriter()
		if oSize, err := io.Copy(nIo, sFile); err != nil {
			return err
		} else {
			myDrives[drive].UsedSize += uint64(oSize)
		}

		sFile.Close()
		oFile.Close()

		x.SrcSha = mIo.Sha256SumToString()

		var vDestShaErr error
		fmt.Println("\r")
		log.Println("Verifying hash of destination file...")
		x.DestSha, vDestShaErr = verifyDestinationFileSHA(x)

		if err := writeJson(&x); err != nil {
			return err
		}

		if vDestShaErr != nil {
			return err
		} else {
			log.Println("Destination hash is good! :)")
		}
	}
	return nil
}

func NewID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80

	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

func main() {
	WalkFunc := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		rName, err2 := NewID()
		if err2 != nil {
			return err2
		}
		f := File{
			ID:      rName,
			Name:    info.Name(),
			Path:    path,
			Size:    uint64(info.Size()),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			Owner:   info.Sys().(*syscall.Stat_t).Uid,
			Group:   info.Sys().(*syscall.Stat_t).Gid,
		}
		FILES = append(FILES, f)
		return err
	}

	log.SetLevel(log.LEVEL_INFO)

	log.Printf("Total backup space %s on %d drives\n",
		humanize.IBytes(calcTotalBackupSpace()), len(myDrives))

	err := filepath.Walk(BACKUP_PATH, WalkFunc)
	if err != nil {
		log.Criticalln(err)
		os.Exit(1)
	}

	log.Printf("Number of Files: %d, Size: %s\n", len(FILES),
		humanize.IBytes(calcTotalDataSize()))

	if err := WriteStuff(); err != nil {
		log.Criticalln(err)
		os.Exit(1)
	}
}
