package core

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type FileType int

const (
	FILE FileType = iota
	DIRECTORY
	SYMLINK
)

type File struct {
	Name           string      `json:"name"`
	Path           string      `json:"path"`
	FileType       FileType    `json:"fileType"`
	DestPath       string      `json:"destPath"`
	Size           uint64      `json:"size"`
	Mode           os.FileMode `json:"mode"`
	ModTime        time.Time   `json:"modTime"`
	AccTime        time.Time   `json:"accessTime"`
	ChgTime        time.Time   `json:"changeTime"`
	Owner          int         `json:"owner"`
	Group          int         `json:"group"`
	SrcSha1        string      `json:"srcSha1"`
	SplitStartByte uint64      `json:"splitStartByte"`
	SplitEndByte   uint64      `json:"splitEndByte"`
}

// VerifyHash checks the sum of the destination file with that of the source
// file. If the hashes differ, then an error is returned.
func (f *File) VerifyHash(file File) (string, error) {
	dFile, err := os.Open(file.Path)
	defer dFile.Close()
	if err != nil {
		return "", err
	}
	s := sha1.New()
	if dSize, err := io.Copy(s, dFile); err != nil {
		return "", err
	} else if dSize != int64(file.Size) {
		return "", fmt.Errorf("A problem occurred reading %q! "+
			"Source Size (%d) != Destination Size (%d)",
			file.Path, file.Size, dSize)
	}
	dSha := hex.EncodeToString(s.Sum(nil))
	if file.SrcSha1 != dSha {
		return "", fmt.Errorf("Source sha1 (%s) != Dest sha1 (%s)",
			file.SrcSha1, dSha)
	}
	return dSha, nil
}

type FileList []File

func NewFileList(c *Context) (FileList, error) {
	bfl := FileList{}
	WalkFunc := func(p string, info os.FileInfo, err error) error {
		if info.IsDir() && p == c.BackupPath && p[len(p)-1] == '/' {
			return nil
		}
		f := File{
			Name:    info.Name(),
			Path:    p,
			Size:    uint64(info.Size()),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			AccTime: time.Unix(info.Sys().(*syscall.Stat_t).Atim.Unix()),
			ChgTime: time.Unix(info.Sys().(*syscall.Stat_t).Ctim.Unix()),
			Owner:   int(info.Sys().(*syscall.Stat_t).Uid),
			Group:   int(info.Sys().(*syscall.Stat_t).Gid),
		}
		if info.IsDir() {
			f.FileType = DIRECTORY
		} else if info.Mode()&os.ModeSymlink != 0 {
			f.FileType = SYMLINK
		}
		bfl = append(bfl, f)
		if err != nil {
			return fmt.Errorf("NewFileList WalkFunc: %s", err.Error())
		}
		return err
	}
	err := filepath.Walk(c.BackupPath, WalkFunc)
	if err != nil {
		return nil, fmt.Errorf("NewFileList: %s", err.Error())
	}
	return bfl, nil
}

type BadFileMetadatError struct {
	Info      *File
	JsonError error
}

func (e BadFileMetadatError) Error() string {
	return fmt.Sprintf("%s\n\n%s\n", e.JsonError, spd.Sdump(e.Info))
}

func (f *FileList) MarshalJSON() ([]byte, error) {
	toJson := func(f *File) ([]byte, error) {
		var v []byte
		v, jErr := json.Marshal(f)
		if jErr != nil {
			f.ModTime = time.Now()
			var err2 error
			v, err2 = json.Marshal(f)
			if err2 != nil {
				return nil, BadFileMetadatError{f, err2}
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
	return jOut, nil
}

func (f *FileList) TotalDataSize() uint64 {
	var totalDataSize uint64
	for _, x := range *f {
		totalDataSize += x.Size
	}
	return totalDataSize
}
