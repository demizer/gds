package core

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/demizer/go-humanize"
)

var spd = spew.ConfigState{Indent: "\t"} //, DisableMethods: true}

type NotEnoughStorageSpaceError struct {
	FileListSize   uint64
	DevicePoolSize uint64
}

func (e NotEnoughStorageSpaceError) Error() string {
	return fmt.Sprintf("Not enough storage space available. Files: %s Device Storage: %s",
		humanize.IBytes(e.FileListSize), humanize.IBytes(e.DevicePoolSize))
}

func checkDevicePoolSpace(f FileList, d DeviceList) error {
	fsize := f.TotalDataSize()
	dsize := d.DevicePoolSize()
	if fsize > dsize {
		return NotEnoughStorageSpaceError{fsize, dsize}
	}
	return nil
}
