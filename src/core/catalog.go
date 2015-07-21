package core

import (
	"path/filepath"
	"strings"
)

type Catalog map[string][]*File

func NewCatalog(b string, d DeviceList, f *FileList) Catalog {
	var dSize uint64
	t := make(Catalog)
	dNum := 0
	for fx, fy := range *f {
		if (dSize + fy.Size) <= d[dNum].SizeBytes {
			dSize += fy.Size
		} else {
			dNum += 1
		}
		key := d[dNum].Name
		t[key] = append(t[key], &(*f)[fx])
		t[key][len(t[key])-1].DestPath = filepath.Join(d[dNum].MountPoint, strings.Replace(fy.Path, b, "", 1))
	}
	return t
}
