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
	err            error
}

// DestPathSha1Sum checks the sum of the destination file with that of the source file. If the hashes differ, then an error
// is returned.
func (f *File) DestPathSha1Sum() (string, error) {
	dFile, err := os.Open(f.DestPath)
	defer dFile.Close()
	if err != nil {
		return "", err
	}
	s := sha1.New()
	if dSize, err := io.Copy(s, dFile); err != nil {
		return "", err
	} else if dSize != int64(f.Size) {
		return "", fmt.Errorf("Error reading file=%q expect_bytes=%d got_bytes=%d", f.DestPath, f.Size, dSize)
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

// GetFileByName return a pointer to the named file.
func (f *FileList) GetFileByName(name string) *File {
	for xx, xy := range *f {
		if xy.Name == name {
			return &(*f)[xx]
		}
	}
	return nil
}
