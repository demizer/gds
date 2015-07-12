package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	log "gopkg.in/inconshreveable/log15.v2"
)

type File struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	Size      uint64      `json:"size"`
	Mode      os.FileMode `json:"mode"`
	ModTime   time.Time   `json:"modTime"`
	Owner     uint32      `json:"owner"`
	Group     uint32      `json:"group"`
	SrcSha    string      `json:"srcSha"`
	DestDrive string      `json:"destDrive"`
	IsDir     bool        `json:"isDir"`
}

// VerifyHash checks the sum of the destination file with that of the source
// file. If the hashes differ, then an error is returned.
func (f *File) VerifyHash(file File) (string, error) {
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

type FileList []File

func NewFileList(path string) (*FileList, error) {
	bfl := FileList{}
	// s := 0
	// t := STATE.Config.Drives
	// d := 0
	// spd.Dump(t)
	WalkFunc := func(p string, info os.FileInfo, err error) error {
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
			Path:    p,
			Size:    uint64(info.Size()),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			Owner:   info.Sys().(*syscall.Stat_t).Uid,
			Group:   info.Sys().(*syscall.Stat_t).Gid,
			// DestPath: filepath.Join(DEST_PATH, rName),
		}
		bfl = append(bfl, f)
		return err
	}
	err := filepath.Walk(path, WalkFunc)
	if err != nil {
		return nil, err
	}
	return &bfl, nil
}

func (f *FileList) MarshalJSON() ([]byte, error) {
	// var b bytes.Buffer
	toJson := func(f *File) ([]byte, error) {
		var v []byte
		v, jErr := json.Marshal(f)
		if jErr != nil {
			// logs.Errorln("An error occurred transcoding to JSON!")
			// logs.Errorln(jErr)
			// logs.Errorf("%#v\n", f)
			// logs.Errorln("Setting time to now and trying again")
			// Sometimes the mod time is not formatted correctly
			f.ModTime = time.Now()
			var err2 error
			v, err2 = json.Marshal(f)
			if err2 != nil {
				// logs.Debugln("IN HERE")
				return nil, &BadFileMetadatError{f, err2}
			}
		}
		return v, nil
	}
	var errs []error
	var jOut []byte
	jOut = append(jOut, '[')
	for _, x := range *f {
		j, err := toJson(&x)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		jOut = append(jOut, j...)
		jOut = append(jOut, ',')
	}
	jOut[len(jOut)-1] = ']'
	// fmt.Println(string(jOut))
	// fmt.Println(errs)
	// fmt.Println("EXITINIGINIGIN")
	// os.Exit(1)
	return jOut, nil
}

func (f *FileList) TotalDataSize() uint64 {
	var totalDataSize uint64
	for _, x := range *f {
		totalDataSize += x.Size
	}
	return totalDataSize
}

// catalog is used to determine the destination drive of the files. A best
// effort is made to not split the files between drives. If the file is too
// large for a single drive, then it is split across drives.
func (f *FileList) catalog(d DriveList) {

}

// Sync synchornises files from the BackupPath to the destination drives. Sync
// will copy new files, delete old files, and fix or update files on the
// destination drive that do not match the source sha1 hash.
func (f *FileList) Sync(d DriveList) {
	for _, y := range *f {
		log.Debug("test", "y", y)
	}
	// drive = backupDrives.AvailableSpace(drive, x)
	// mIo := NewIoReaderWriter(oFile, x.Size)
	// logs.Printf("Writing %q (%s)...\n\r",
	// x.Name, humanize.IBytes(x.Size))
	// nIo := mIo.MultiWriter()
	// if oSize, err := io.Copy(nIo, sFile); err != nil {
	// return err
	// } else {
	// backupDrives[drive].UsedSize += uint64(oSize)
	// }
	// sFile.Close()
	// x.SrcSha = mIo.Sha256SumToString()
	// var vDestShaErr error
	// fmt.Println("\r")
	// logs.Println("Verifying hash of destination file...")
	// x.DestSha, vDestShaErr = verifyDestinationFileSHA(x)
	// if err := writeJson(&x); err != nil {
	// return err
	// }
	// if vDestShaErr != nil {
	// return err
	// } else {
	// logs.Println("Destination hash is good! :)")
	// }
	// }
}
