package core

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type FileType int

const (
	FILE FileType = iota
	DIRECTORY
	SYMLINK
)

type FileNotFoundError int

func (e FileNotFoundError) Error() string {
	return "File not found"
}

type FileSourceNotReadable struct {
	FilePath  string
	ReadError string
}

func (e FileSourceNotReadable) Error() string {
	return fmt.Sprintf("Could not read %q, %s", e.FilePath, e.ReadError)
}

type FileBadMetadataError struct {
	Info      *File
	JsonError error
}

func (e FileBadMetadataError) Error() string {
	return fmt.Sprintf("%s\n\n%s\n", e.JsonError, spd.Sdump(e.Info))
}

type BadDestPathSha1Sum struct {
	srcSha1sum  string
	destSha1sum string
}

func (e BadDestPathSha1Sum) Error() string {
	return fmt.Sprintf("Destination file sum mismatch: expect_sha1sum=%s got=%s", e.srcSha1sum, e.destSha1sum)
}

// File describes a file being stored on a device.
type File struct {
	Name          string   `json:"name"`
	Path          string   `json:"path"`
	Size          uint64   `json:"size"`
	Sha1Sum       string   `json:"sha1Sum"`
	FileType      FileType `json:"fileType"`
	SymlinkTarget string   `json:"symlinkTarget"`

	// File metadata
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"modTime"`
	AccTime time.Time   `json:"accessTime"`
	ChgTime time.Time   `json:"changeTime"`
	Owner   int         `json:"owner"`
	Group   int         `json:"group"`

	// A destination file can be split across multiple devices
	DestFiles []*DestFile
}

// IsSplit returns true if the file is split across devices.
func (f *File) IsSplit() bool {
	if len(f.DestFiles) > 1 {
		return true
	}
	return false
}

// Add a destination file record to the file index
func (f *File) AddDestFile(file *DestFile) {
	remain := f.Size - file.EndByte
	Log.Infof("%q adding %q size: %d total: %d remain: %d", file.DeviceName, f.Name,
		file.Size, f.Size, remain)
	f.DestFiles = append(f.DestFiles, file)
}

func (f *File) SetSymlinkTargetPath() (err error) {
	f.SymlinkTarget, err = filepath.EvalSymlinks(f.Path)
	return
}

// DestPathSha1Sum checks the sum of the destination file with that of the source file. If the hashes differ, then an error
// is returned.
func (f *File) DestPathSha1Sum() (err error) {
	for _, file := range f.DestFiles {
		var s = sha1.New()
		var df *os.File
		if len(f.Name) > 0 {
			if df, err = os.Open(file.Path); err != nil {
				return
			}
		}
		if _, err = io.Copy(s, df); err != nil {
			return
		}
		ds := hex.EncodeToString(s.Sum(nil))
		if f.Sha1Sum != ds {
			return fmt.Errorf("sha1sum error: expect: %s got: %s", f.Sha1Sum, ds)
		}
	}
	return
}
