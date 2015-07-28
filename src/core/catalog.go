package core

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	log "gopkg.in/inconshreveable/log15.v2"
)

type Catalog map[string][]*File

// duplicateDirTree is used when a file is split across devices. The parent directories of the file must be duplicated on the
// next device. Copies of the individual parent directories of file f are copied from the catalog and appended to the new
// catalog device.
func duplicateDirTree(c *Catalog, d *Device, bpath string, f *File) uint64 {
	ppaths := strings.Split(path.Dir(f.DestPath[len(d.MountPoint):]), string(os.PathSeparator))
	var usize uint64
	for _, xy := range ppaths {
	nextPath:
		for _, yy := range *c {
			for _, zy := range yy {
				if zy.Name == xy {
					nf := *zy
					nf.DestPath = path.Join(d.MountPoint, zy.DestPath[len(d.MountPoint):])
					nf.Size = zy.Size
					usize += nf.Size
					(*c)[d.Name] = append((*c)[d.Name], &nf)
					break nextPath
				}
			}
		}
	}
	return usize
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
	for fx, _ := range c.Files {
		split := false
		f := &(c.Files)[fx]
		d := c.Devices[dNum]
		if (dSize + f.Size) <= d.SizeBytes {
			dSize += f.Size
		} else if (dSize+c.SplitMinSize) <= d.SizeBytes && f.Size > d.SizeBytes-d.UsedSize {
			// Split de file
			split = true
			f.SplitStartByte = 0
			f.SplitEndByte = c.Devices[dNum].SizeBytes - dSize
			log.Debug("Splitting file", "file_name", f.Name, "file_size", f.Size, "file_split_start_byte",
				f.SplitStartByte, "file_split_end_byte", f.SplitEndByte, "device_used + splitMinSize",
				dSize+c.SplitMinSize, "device_size_bytes", d.SizeBytes, "device_number", dNum)
		} else {
			dNum += 1
			dSize = 0
		}

		if f.Path == "/dev/zero" {
			// Used only for testing
			continue
		}

		f.DestPath = filepath.Join(d.MountPoint, f.Path[len(bpath):])
		if split {
			t[d.Name] = append(t[d.Name], f)
			dNum += 1
			dSize = 0
			lastf := *f
			for {
				// Loop filling up devices as needed
				d = c.Devices[dNum]
				fNew := lastf
				fNew.SplitStartByte = fNew.SplitEndByte + 1
				fNew.DestPath = filepath.Join(d.MountPoint, fNew.Path[len(bpath):])
				dSize += duplicateDirTree(&t, &d, bpath, &fNew)
				fNew.SplitEndByte = fNew.Size
				log.Debug("File/Device state in split", "file_remain_size", fNew.Size-fNew.SplitStartByte,
					"device_usage", dSize, "device_name", d.Name, "device_size", d.SizeBytes)
				if (fNew.Size - fNew.SplitStartByte) > (d.SizeBytes - dSize) {
					fNew.SplitEndByte = fNew.SplitStartByte + (d.SizeBytes - dSize)
				}
				t[d.Name] = append(t[d.Name], &fNew)
				if fNew.SplitEndByte == fNew.Size {
					break
				}
				dNum += 1
				dSize = 0
				if dNum == len(c.Devices) {
					break
				}
				lastf = fNew
			}
		} else {
			t[d.Name] = append(t[d.Name], f)
		}

	}
	return t
}
