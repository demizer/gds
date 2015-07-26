package core

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Catalog map[string][]*File

// duplicateDirTree is used when a file is split across devices. The parent directories of the file must be duplicated on the
// next device. Copies of the individual parent directories of file f are copied from the catalog and appended to the new
// catalog device.
func duplicateDirTree(c *Catalog, d *Device, bpath string, f *File) {
	ppaths := strings.Split(path.Dir(f.DestPath[len(d.MountPoint):]), string(os.PathSeparator))
	for _, xy := range ppaths {
	nextPath:
		for _, yy := range *c {
			for _, zy := range yy {
				if zy.Name == xy {
					nf := *zy
					nf.DestPath = path.Join(d.MountPoint, zy.DestPath[len(d.MountPoint):])
					(*c)[d.Name] = append((*c)[d.Name], &nf)
					break nextPath
				}
			}
		}
	}
}

// NewCatalog returns a new catalog. Files are matched to a device in order. When a catalog entry is made, the destination
// path is also calculated. NewCatalog assumes all files will fit in the storage pool.
func NewCatalog(c *Context) Catalog {
	var dSize uint64
	t := make(Catalog)
	dNum := 0
	bpath := c.BackupPath
	if c.BackupPath[len(c.BackupPath)-1] != '/' {
		bpath = path.Dir(c.BackupPath)
	}
	for fx, fy := range c.Files {
		split := false
		if (dSize + fy.Size) <= c.Devices[dNum].SizeBytes {
			dSize += fy.Size
		} else if (dSize + c.SplitMinSize) <= c.Devices[dNum].SizeBytes {
			// Split de file
			split = true
			(c.Files)[fx].SplitStartByte = 0
			(c.Files)[fx].SplitEndByte = c.Devices[dNum].SizeBytes - dSize
		} else {
			dNum += 1
			dSize = 0
		}
		key := c.Devices[dNum].Name
		t[key] = append(t[key], &(c.Files)[fx])

		if fy.Path == "/dev/zero" {
			// For testing
			continue
		}

		if !split {
			t[key][len(t[key])-1].DestPath = filepath.Join(c.Devices[dNum].MountPoint, fy.Path[len(bpath):])
		} else {
			// Current device
			t[key][len(t[key])-1].DestPath = filepath.Join(c.Devices[dNum].MountPoint, fy.Path[len(bpath):])

			// Next device
			dNum += 1
			key = c.Devices[dNum].Name
			fc := (c.Files)[fx]
			fc.SplitStartByte = fc.SplitEndByte + 1
			fc.SplitEndByte = fc.Size
			duplicateDirTree(&t, &c.Devices[dNum], bpath, &fc)
			t[key] = append(t[key], &fc)
			t[key][len(t[key])-1].DestPath = filepath.Join(c.Devices[dNum].MountPoint, fy.Path[len(bpath):])
			dSize += fc.Size - fc.SplitEndByte
		}
	}
	return t
}
