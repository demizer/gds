package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

var FILES FileList

type File struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Size     uint64      `json:"size"`
	Mode     os.FileMode `json:"mode"`
	ModTime  time.Time   `json:"modTime"`
	Owner    uint32      `json:"owner"`
	Group    uint32      `json:"group"`
	SrcSha   string      `json:"srcSha"`
	DestPath string      `json:"destPath"`
	DestSha  string      `json:"destSha"`
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

func (f *FileList) MarshalJson() ([]byte, error) {
	// var b bytes.Buffer
	for x := range *f {
		fmt.Println(x)
	}
	os.Exit(1)
	return nil, nil
}

func (f *FileList) TotalDataSize() uint64 {
	var totalDataSize uint64
	for _, x := range *f {
		totalDataSize += x.Size
	}
	return totalDataSize
}

func NewFileList() (*FileList, error) {
	bfl := FileList{}
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
		bfl = append(bfl, f)
		return err
	}
	err := filepath.Walk(BACKUP_PATH, WalkFunc)
	if err != nil {
		return nil, err
	}
	return &bfl, nil
}
