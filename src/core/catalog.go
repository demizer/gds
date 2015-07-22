package core

import (
	"path"
	"path/filepath"
	"strings"
)

type Catalog map[string][]*File

// NewCatalog returns a new catalog. Files are matched to a device in order. When a catalog entry is made, the destination
// path is also calculated. NewCatalog assumes all files will fit in the storage pool.
func NewCatalog(backupPath string, d DeviceList, f *FileList) Catalog {
	var dSize uint64
	t := make(Catalog)
	dNum := 0
	bpath := backupPath
	if backupPath[len(backupPath)-1] != '/' {
		bpath = path.Dir(backupPath)
	}
	for fx, fy := range *f {
		if (dSize + fy.Size) <= d[dNum].SizeBytes {
			dSize += fy.Size
		} else {
			dNum += 1
		}
		key := d[dNum].Name
		t[key] = append(t[key], &(*f)[fx])
		t[key][len(t[key])-1].DestPath = filepath.Join(d[dNum].MountPoint,
			strings.Replace(fy.Path, bpath, "", 1))
	}
	return t
}
