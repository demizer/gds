package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/demizer/go-elog"
	"github.com/dustin/go-humanize"
)

var spd = spew.ConfigState{Indent: "\t"} //, DisableMethods: true}

const BACKUP_PATH = "/mnt/data"
const DEST_PATH = "/mnt/backup"
const PADDING = 10485760 // 10G

var backupDrives = DriveList{
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

func WriteStuff() error {
	mdName := filepath.Join(os.Getenv("HOME"), "bigdata_files.json")
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
		drive = backupDrives.AvailableSpace(drive, x)

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
			backupDrives[drive].UsedSize += uint64(oSize)
		}

		sFile.Close()
		oFile.Close()

		x.SrcSha = mIo.Sha256SumToString()

		// var vDestShaErr error
		fmt.Println("\r")
		// log.Println("Verifying hash of destination file...")
		// x.DestSha, vDestShaErr = verifyDestinationFileSHA(x)

		if err := writeJson(&x); err != nil {
			return err
		}

		// if vDestShaErr != nil {
		// return err
		// } else {
		// log.Println("Destination hash is good! :)")
		// }
	}
	return nil
}

func main() {
	log.SetLevel(log.LEVEL_INFO)

	log.Printf("Total backup space %s on %d drives\n",
		humanize.IBytes(FILES.TotalDataSize()), len(backupDrives))

	files, err := NewFileList()
	if err != nil {
		log.Criticalln(err)
		os.Exit(1)
	}

	log.Printf("Number of Files: %d, Size: %s\n", len(FILES),
		humanize.IBytes(files.TotalDataSize()))

	if err := WriteStuff(); err != nil {
		log.Criticalln(err)
		os.Exit(1)
	}
}
