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

type File struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	DestPath string      `json:"destPath"`
	Size     uint64      `json:"size"`
	Mode     os.FileMode `json:"mode"`
	ModTime  time.Time   `json:"modTime"`
	Owner    uint32      `json:"owner"`
	Group    uint32      `json:"group"`
	SrcSha1  string      `json:"srcSha1"`
	IsDir    bool        `json:"isDir"`
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

func NewFileList(path string) (*FileList, error) {
	bfl := FileList{}
	WalkFunc := func(p string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		f := File{
			Name:    info.Name(),
			Path:    p,
			Size:    uint64(info.Size()),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			Owner:   info.Sys().(*syscall.Stat_t).Uid,
			Group:   info.Sys().(*syscall.Stat_t).Gid,
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
	toJson := func(f *File) ([]byte, error) {
		var v []byte
		v, jErr := json.Marshal(f)
		if jErr != nil {
			f.ModTime = time.Now()
			var err2 error
			v, err2 = json.Marshal(f)
			if err2 != nil {
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
	return jOut, nil
}

func (f *FileList) TotalDataSize() uint64 {
	var totalDataSize uint64
	for _, x := range *f {
		totalDataSize += x.Size
	}
	return totalDataSize
}