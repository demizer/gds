package core

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/Sirupsen/logrus"
)

// DestFile describes a destination file.
type DestFile struct {
	DeviceName string
	Path       string
	Size       uint64
	StartByte  uint64
	EndByte    uint64
	Sha1Sum    string
	err        error // Used to record errors that occurr when creating or writing to the dest file.
}

// NewDestFile will return a new destination file with the UUID dest path set. If df is not nil, then the start and end bytes
// of the new dest file will be based off the start and end bytes of the previous dest file (pvf).
func NewDestFile(f *File, d *Device, pvf *DestFile, df *DestFile) *DestFile {
	fp := &DestFile{
		DeviceName: d.Name,
		Size:       f.Size,
		EndByte:    f.Size,
	}
	if pvf != nil {
		fp.StartByte = pvf.EndByte
	}
	fp.generateDestPath(d.MountPoint)
	return fp
}

// generateDestPath will generate a new UUID destination path for the destination file using mp as the mount point.
func (df *DestFile) generateDestPath(mp string) (err error) {
	gid, err := NewID()
	if err != nil {
		return
	}
	df.Path = filepath.Join(mp, gid)
	return
}

// setMetaData sets permissions of the destination file.
func (df *DestFile) setMetaData(f *File) error {
	var err error
	mTimeval := syscall.NsecToTimespec(f.ModTime.UnixNano())
	times := []syscall.Timespec{
		mTimeval,
		mTimeval,
	}
	// err = os.Chown(f.Source.Path, f.Source.Owner, f.Source.Group)
	err = os.Chown(df.Path, f.Owner, f.Group)
	if err == nil {
		Log.WithFields(logrus.Fields{"owner": f.Owner, "group": f.Group}).Debugln("Set owner")
		// Change the modtime of a symlink without following it
		err = LUtimesNano(df.Path, times)
		if err == nil {
			Log.WithFields(logrus.Fields{"modTime": f.ModTime}).Debugln("Set modification time")
		}
	}
	if err != nil {
		return fmt.Errorf("setMetaData: %s", err.Error())
	}
	return nil
}

// createFile is a helper function for creating directories, symlinks, and regular files. If it encounters errors creating
// these files, the error is sent on the cerr buffered error channel.
func (df *DestFile) createFile(f *File) {
	var err error
	if f.Owner != os.Getuid() && os.Getuid() != 0 {
		df.err = SyncIncorrectOwnershipError{f.Path, f.Owner, os.Getuid()}
		Log.Errorf("createFile: %s", df.err)
		return
	}
	var oFile *os.File
	if _, lerr := os.Stat(df.Path); lerr != nil {
		oFile, err = os.Create(df.Path)
		err = oFile.Close()
		if err == nil {
			Log.WithFields(logrus.Fields{"name": f.Name}).Debugln("Created empty file")
		}
	}
	if err == nil {
		err = df.setMetaData(f)
		if err != nil {
			df.err = fmt.Errorf("createFile: %s", err.Error())
		}
	}
}

// source returns the parent file of the destination file
func (df *DestFile) source(fi *FileIndex) *File {
	for _, f := range *fi {
		for _, df2 := range f.DestFiles {
			if df == df2 {
				return f
			}
		}
	}
	return nil
}
