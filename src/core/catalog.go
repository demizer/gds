package core

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
)

// Catalog is the data structure that indicates onto which device files in the backup will be copied to. The Catalog is also
// saved as the record of the backup. The key is the name of the device, and the values are a list of pointers to File
// structs. Using a map of pre-determined sync paths makes it possible to sync to multiple devices at once, concurrently.
type Catalog map[string][]*File

// duplicateDirTree is used when a file is split across devices, or a file in a directory tree is destined for a fresh
// device. The directory tree of the file must be duplicated on the next device within the catalog.
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
	dNum := 0
	t := make(Catalog)

	bpath := c.BackupPath
	if c.BackupPath[len(c.BackupPath)-1] != '/' {
		bpath = path.Dir(c.BackupPath)
	}

	for fx, _ := range c.Files {
		split := false
		f := &(c.Files)[fx]
		d := c.Devices[dNum]
		if (dSize + f.Size) <= d.Size {
			// Add the file to the current device
			dSize += f.Size
		} else if (dSize+c.SplitMinSize) <= d.Size && f.Size > d.Size-d.UsedSize {
			// Split de file, more logic to follow ...
			split = true
			f.SplitStartByte = 0
			f.SplitEndByte = c.Devices[dNum].Size - dSize
			Log.WithFields(logrus.Fields{
				"file_name":                  f.Name,
				"file_size":                  f.Size,
				"file_split_start_byte":      f.SplitStartByte,
				"file_split_end_byte":        f.SplitEndByte,
				"device_used_+_splitMinSize": dSize + c.SplitMinSize,
				"device_size_bytes":          d.Size,
				"device_number":              dNum,
			}).Debugln("NewCatalog: Splitting file")
		} else {
			// Out of device space, get the next device
			dNum += 1
			d = c.Devices[dNum]
			dSize = 0
		}

		f.DestPath = filepath.Join(d.MountPoint, f.Path[len(bpath):])
		if split {
			t[d.Name] = append(t[d.Name], f)
			dNum += 1
			dSize = 0
			lastf := *f
			for {
				// Loop until the file is completely accounted for, across devices if necessary
				d = c.Devices[dNum]
				fNew := lastf
				fNew.SplitStartByte = fNew.SplitEndByte + 1
				fNew.SplitEndByte = fNew.Size
				fNew.DestPath = filepath.Join(d.MountPoint, fNew.Path[len(bpath):])

				// Duplicate the dir tree to of the file on the new device
				dSize += duplicateDirTree(&t, &d, bpath, &fNew)

				Log.WithFields(logrus.Fields{
					"file_remain_size": fNew.Size - fNew.SplitStartByte,
					"device_usage":     dSize,
					"device_name":      d.Name,
					"device_size":      d.Size}).Debug("NewCatalog: File/Device state in split")

				// If the file is still larger than the new divice, use all of the available space
				if (fNew.Size - fNew.SplitStartByte) > (d.Size - dSize) {
					fNew.SplitEndByte = fNew.SplitStartByte + (d.Size - dSize)
				}

				Log.WithFields(logrus.Fields{
					"file_name":                  fNew.Name,
					"file_size":                  fNew.Size,
					"file_split_start_byte":      fNew.SplitStartByte,
					"file_split_end_byte":        fNew.SplitEndByte,
					"device_used_+_splitMinSize": dSize + c.SplitMinSize,
					"device_size_bytes":          d.Size,
					"device_number":              dNum,
				}).Debugln("NewCatalog: Splitting file")

				t[d.Name] = append(t[d.Name], &fNew)
				if fNew.SplitEndByte == fNew.Size {
					// The file is accounted for, break the loop
					break
				}

				// If the exec path reaches this point, we are out of device space, but still have a portion
				// of file remaning. Increase the device number, we'll set it in the next loop.
				dNum += 1
				dSize = 0
				if dNum == len(c.Devices) {
					// Out of devices
					break
				}
				lastf = fNew
			}
		} else {
			if dSize == 0 {
				// A new device has been added above, and the file is not being split. If the file is burried
				// in a directory tree, we need to add those directories to the catalog before setting the
				// file otherwise the file copy will fail.
				dSize += duplicateDirTree(&t, &d, bpath, f)
			}
			t[d.Name] = append(t[d.Name], f)
		}
	}
	return t
}
