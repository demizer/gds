package core

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/demizer/go-humanize"
)

// CatalogNotEnoughDevicePoolSpaceError is an error given when the backup size exceeds the device pool storage size.
type CatalogNotEnoughDevicePoolSpaceError struct {
	TotalCatalogSize    uint64
	TotalDevicePoolSize uint64
}

// Error implements the Error interface.
func (e CatalogNotEnoughDevicePoolSpaceError) Error() string {
	return fmt.Sprintf("Inadequate device pool space! TotalCatalogSize: %d (%s) TotalDevicePoolSize: %d (%s)",
		e.TotalCatalogSize, humanize.IBytes(e.TotalCatalogSize), e.TotalDevicePoolSize,
		humanize.IBytes(e.TotalDevicePoolSize))
}

// Catalog is the data structure that indicates onto which device files in the backup will be copied to. The Catalog is also
// saved as the record of the backup. The key is the name of the device, and the values are a list of pointers to File
// structs. Using a map of pre-determined sync paths makes it possible to sync to multiple devices at once, concurrently.
type Catalog map[string][]*File

// duplicateDirTree is used when a file is split across devices, or a file in a directory tree is destined for a fresh
// device. The directory tree of the file must be duplicated on the next device within the catalog.
// func duplicateDirTree(c *Catalog, d *Device, bpath string, f *File) uint64 {
// ppaths := strings.Split(path.Dir(f.DestPath[len(d.MountPoint):]), string(os.PathSeparator))
// var usize uint64
// for _, pathBase := range ppaths {
// var found *File
// for _, files := range *c {
// for _, file := range files {
// if file.Name == pathBase {
// found = file
// }
// }
// }
// if found != nil {
// nf := *found
// nf.DestPath = path.Join(d.MountPoint, found.DestPath[len(d.MountPoint):])
// // spd.Dump(nf.DestSize)
// nf.DestSize = d.BlockSize
// usize += nf.DestSize
// Log.WithFields(logrus.Fields{
// "nf.Name":     nf.Name,
// "nf.DestPath": nf.DestPath,
// "d.Name":      d.Name,
// }).Debugln("Add directory to device")
// // spd.Dump(nf)
// (*c)[d.Name] = append((*c)[d.Name], &nf)
// }
// }
// return usize
// }

// NewCatalog returns a new catalog. Files are matched to a device in order. When a catalog entry is made, the destination
// path is also calculated. NewCatalog assumes all files will fit in the storage pool.
func NewCatalog(c *Context) (Catalog, error) {
	var err error
	var dSize uint64
	var notEnoughSpaceError bool
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
		f.DestSize = d.BlockSize
		if f.Name == "ROMS" {
			Log.Debugln("poop", spd.Sdump(f))
		}
		if f.FileType == FILE {
			f.DestSize = f.SourceSize
		} else if f.FileType == DIRECTORY && f.SourceSize > d.BlockSize {
			Log.Error("IN HERE YO")
			f.DestSize = f.SourceSize
		}
		if dSize == 0 {
			Log.WithFields(logrus.Fields{
				"d.SizeTotal":  d.SizeTotal,
				"deviceNumber": dNum,
			}).Debugln("Device stats")
		}
		Log.WithFields(logrus.Fields{
			"f.Name":       f.Name,
			"f.SourceSize": f.SourceSize,
			"f.DestSize":   f.DestSize,
			"f.FileType":   f.FileType,
			"dSize":        dSize,
		}).Debugln("NewCatalog: FILE")
		if (dSize + f.DestSize) <= d.SizeTotal {
			// Log.Debugln("File size + dSize is less than or equal to device total size")
			Log.Debugln("Adding file to device!")
			dSize += f.DestSize
		} else if (dSize+c.SplitMinSize) <= d.SizeTotal && f.SourceSize > d.SizeTotal-dSize {
			Log.Debugln("Splitting file!")
			// Split de file, more logic to follow ...
			split = true
			f.SplitStartByte = 0
			f.SplitEndByte = c.Devices[dNum].SizeTotal - dSize
			f.DestSize = f.SplitEndByte - f.SplitStartByte
		} else {
			// Out of device space, get the next device
			Log.Debugln("Out of device space!")
			if dNum+1 == len(c.Devices) {
				Log.Error("Total backup size is greater than device pool size!")
				notEnoughSpaceError = true
				c.Devices = append(c.Devices, Device{
					Name:       "overrun",
					MountPoint: "none",
					SizeTotal:  d.SizeTotal,
				})
			} else {
				dNum += 1
				Log.Debugf("NewCatalog: Using c.Devices[%d]", dNum)
				d = c.Devices[dNum]
				dSize = 0
			}
		}
		Log.WithFields(logrus.Fields{
			"dSize":       dSize,
			"d.SizeTotal": d.SizeTotal,
			"f.DestSize":  f.DestSize,
		}).Debugln("NewCatalog: After size calc")
		f.DestPath = filepath.Join(d.MountPoint, f.Path[len(bpath):])
		if split {
			Log.WithFields(logrus.Fields{
				"d.Name":           d.Name,
				"d.SizeTotal":      d.SizeTotal,
				"f.Name":           f.Name,
				"f.DestSize":       f.DestSize,
				"f.SplitStartByte": f.SplitStartByte,
				"f.SplitEndByte":   f.SplitEndByte,
				"fileRemaining":    f.SourceSize - f.DestSize,
			}).Debug("NewCatalog: Split File before loop")
			t[d.Name] = append(t[d.Name], f)
			dNum += 1
			dSize = 0
			lastf := *f
			for {
				// Loop until the file is completely accounted for, across devices if necessary
				d = c.Devices[dNum]
				fNew := lastf
				// fmt.Println("1111", fNew.DestSize)

				// Duplicate the dir tree to of the file on the new device
				Log.WithFields(logrus.Fields{
					"dSize":       dSize,
					"d.SizeTotal": d.SizeTotal,
				}).Debug("NewCatalog: Before tree duplication")

				dSize += duplicateDirTree(&t, &d, bpath, &fNew)

				// fmt.Println("2222", fNew.DestSize)
				Log.WithFields(logrus.Fields{
					"dSize": dSize,
				}).Debug("NewCatalog: After tree duplication")

				// fmt.Println("SPD DUMP fNew")
				// spd.Dump(fNew)
				// Log.WithFields(logrus.Fields{
				// "dSize":               dSize,
				// "d.Name":              d.Name,
				// "d.SizeTotal":         d.SizeTotal,
				// "fNew.Name":           fNew.Name,
				// "lastf.DestSize":      lastf.DestSize,
				// "fNew.DestSize":       fNew.DestSize,
				// "fNew.SplitStartByte": fNew.SplitStartByte,
				// "fNew.SplitEndByte":   fNew.SplitEndByte,
				// "fileRemaining":       fNew.SourceSize - fNew.DestSize,
				// }).Debug("NewCatalog: Split File")
				// Setup the new file and determine if we need to split again
				fNew.SplitStartByte = lastf.SplitEndByte + 1
				fNew.DestPath = filepath.Join(d.MountPoint, fNew.Path[len(bpath):])
				fNew.SplitEndByte = fNew.SourceSize
				fNew.DestSize = fNew.SourceSize - fNew.SplitStartByte

				// fmt.Println("3333", fNew.DestSize)

				// If the file is still larger than the new divice, use all of the available space
				if (dSize + fNew.DestSize) >= d.SizeTotal {
					// fmt.Println("sourcesize", fNew.SourceSize, "d.SizeTotal-dSize", (d.SizeTotal - dSize))
					// fNew.SplitEndByte = (fNew.SourceSize - fNew.SplitStartByte) - (d.SizeTotal - dSize) // Use the remaining device space
					fNew.SplitEndByte = fNew.SplitStartByte + (d.SizeTotal - dSize) // Use the remaining device space
					// fmt.Println("splitstartbite", fNew.SplitStartByte, "splitendbyte", fNew.SplitEndByte)
					fNew.DestSize = fNew.SplitEndByte - fNew.SplitStartByte
					// fmt.Println("destsize", fNew.DestSize)
				}
				// fmt.Println("4444", fNew.DestSize)
				dSize += fNew.DestSize
				Log.WithFields(logrus.Fields{
					"dSize":               dSize,
					"d.Name":              d.Name,
					"d.SizeTotal":         d.SizeTotal,
					"fNew.Name":           fNew.Name,
					"fNew.DestSize":       fNew.DestSize,
					"fNew.SplitStartByte": fNew.SplitStartByte,
					"fNew.SplitEndByte":   fNew.SplitEndByte,
					"fileRemaining":       fNew.SourceSize - fNew.SplitEndByte,
				}).Debug("NewCatalog: Split File")

				t[d.Name] = append(t[d.Name], &fNew)
				// if fNew.SplitEndByte == fNew.SourceSize && d.SizeTotal <= (dSize+fNew.DestSize) {
				if fNew.SplitEndByte == fNew.SourceSize {
					// The file is accounted for, break the loop
					Log.Debug("File is accounted for")
					break
				}
				// If the exec path reaches this point, we are out of device space, but still have a portion
				// of file remaning. Increase the device number, we'll set it in the next loop.
				dNum += 1
				if dNum > len(c.Devices)-1 {
					Log.Error("Total backup size is greater than device pool size!")
					notEnoughSpaceError = true
				}

				dSize = 0
				if dNum == len(c.Devices) && !notEnoughSpaceError {
					// Out of devices
					break
				}
				if notEnoughSpaceError {
					// Add a fake device so that we can finish and report an error back with actual usage
					// data.
					c.Devices = append(c.Devices, Device{
						Name:       "overrun",
						MountPoint: "none",
						SizeTotal:  fNew.SourceSize, // With room to spare
					})
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
	if notEnoughSpaceError {
		err = CatalogNotEnoughDevicePoolSpaceError{
			TotalCatalogSize:    t.TotalSize(),
			TotalDevicePoolSize: c.Devices.TotalSize(),
		}
	}

	return t, err
}

// TotalSize returns the real size of the backup. If files are split across devices, the parent directories of the
// file is duplicated on successive devices. This increases the actual total size of the backup.
func (c *Catalog) TotalSize() uint64 {
	var total uint64
	for _, val := range *c {
		for _, f := range val {
			if f.SplitEndByte != 0 {
				total += f.SplitEndByte - f.SplitStartByte
			} else {
				total += f.DestSize
			}
		}
	}
	return total
}
