package core

import (
	"crypto/sha1"
	"encoding/hex"
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

type BadFileMetadatError struct {
	Info      *File
	JsonError error
}

func (e BadFileMetadatError) Error() string {
	return fmt.Sprintf("%s\n\n%s\n", e.JsonError, spd.Sdump(e.Info))
}

type BadDestPathSha1Sum struct {
	srcSha1sum  string
	destSha1sum string
}

func (e BadDestPathSha1Sum) Error() string {
	return fmt.Sprintf("Destination file sum mismatch: expect_sha1sum=%s got=%s", e.srcSha1sum, e.destSha1sum)
}

type File struct {
	Name           string      `json:"name"`
	Path           string      `json:"path"`
	FileType       FileType    `json:"fileType"`
	SourceSize     uint64      `json:"sourceSize"` // The actual file size
	DestPath       string      `json:"destPath"`   // The file size at the destination
	DestSize       uint64      `json:"destSize"`
	Mode           os.FileMode `json:"mode"`
	ModTime        time.Time   `json:"modTime"`
	AccTime        time.Time   `json:"accessTime"`
	ChgTime        time.Time   `json:"changeTime"`
	Owner          int         `json:"owner"`
	Group          int         `json:"group"`
	SrcSha1        string      `json:"srcSha1"`
	SplitStartByte uint64      `json:"splitStartByte"`
	SplitEndByte   uint64      `json:"splitEndByte"`
	err            error
}

// DestPathSha1Sum checks the sum of the destination file with that of the source file. If the hashes differ, then an error
// is returned.
func (f *File) DestPathSha1Sum() (string, error) {
	var dFile *os.File
	var err error
	var s = sha1.New()
	if len(f.Name) > 0 {
		dFile, err = os.Open(f.DestPath)
		defer func() {
			err = dFile.Close()
		}()
		if err != nil {
			return "", err
		}
	}
	if _, err := io.Copy(s, dFile); err != nil {
		return "", err
	}
	dSha := hex.EncodeToString(s.Sum(nil))
	if f.SrcSha1 != dSha {
		return "", fmt.Errorf("sha1sum error: expect_sha1sum=%s got=%s", f.SrcSha1, dSha)
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
			Name:       info.Name(),
			Path:       p,
			SourceSize: uint64(info.Size()),
			Mode:       info.Mode(),
			ModTime:    info.ModTime(),
			AccTime:    time.Unix(info.Sys().(*syscall.Stat_t).Atim.Unix()),
			ChgTime:    time.Unix(info.Sys().(*syscall.Stat_t).Ctim.Unix()),
			Owner:      int(info.Sys().(*syscall.Stat_t).Uid),
			Group:      int(info.Sys().(*syscall.Stat_t).Gid),
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

func (f *FileList) TotalDataSize() uint64 {
	var totalDataSize uint64
	for _, x := range *f {
		totalDataSize += x.DestSize
	}
	return totalDataSize
}

type FileNotFound int

func (e FileNotFound) Error() string {
	return "File not found"
}

// FileByName return a pointer to the named file.
func (f *FileList) FileByName(name string) (*File, error) {
	for xx, xy := range *f {
		if xy.Name == name {
			return &(*f)[xx], nil
		}
	}
	return nil, new(FileNotFound)
}
