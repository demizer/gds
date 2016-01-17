package core

import "path/filepath"

// DestFile describes a destination file.
type DestFile struct {
	Source           *File `json:"-"`
	DeviceName       string
	DeviceMountPoint string
	Path             string
	Size             uint64
	StartByte        uint64
	EndByte          uint64
	Sha1Sum          string
	err              error // Used to record errors that occurr when creating or writing to the dest file.
}

// NewDestFile will return a new destination file with the UUID dest path set. If df is not nil, then the start and end bytes
// of the new dest file will be based off the start and end bytes of the previous dest file (pvf).
func NewDestFile(f *File, pvf *DestFile, df *DestFile, d *Device) *DestFile {
	fp := &DestFile{
		Source:           f,
		DeviceName:       d.Name,
		DeviceMountPoint: d.MountPoint,
		Size:             f.Size,
		EndByte:          f.Size,
	}
	if pvf != nil {
		fp.StartByte = pvf.EndByte
	}
	fp.generateDestPath()
	return fp
}

// generateDestPath will generate a new UUID destination path for the destination file using the device data.
func (d *DestFile) generateDestPath() (err error) {
	gid, err := NewID()
	if err != nil {
		return
	}
	d.Path = filepath.Join(d.DeviceMountPoint, gid)
	return
}
