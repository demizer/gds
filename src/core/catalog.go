package core

import (
	"fmt"
	"path/filepath"

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

// NewCatalog returns a new catalog. Files are matched to a device in order. When a catalog entry is made, the destination
// path is also calculated. NewCatalog assumes all files will fit in the storage pool.
func NewCatalog(c *Context) (Catalog, error) {
	var err error
	var dSize uint64
	var notEnoughSpaceError bool
	dNum := 0
	t := make(Catalog)

	for fx, _ := range c.Files {
		split := false

		f := &(c.Files)[fx]
		if f.FileType == DIRECTORY {
			continue
		}

		d := c.Devices[dNum]
		paddBytes := uint64(float64(c.Devices[dNum].SizeTotal) * (c.PaddingPercentage / 100))
		if len(t[d.Name]) == 1 {
			Log.WithFields(logrus.Fields{
				"d.SizeTotal": d.SizeTotal, "paddBytes": paddBytes, "deviceNumber": dNum,
				"dev.SizeTotal-paddBytes": c.Devices[dNum].SizeTotal - paddBytes,
			}).Debugln("Device stats")
		}

		f.DestSize = f.SourceSize
		Log.WithFields(logrus.Fields{
			"f.Name": f.Name, "f.SourceSize": f.SourceSize, "f.DestSize": f.DestSize,
			"f.FileType": f.FileType, "dSize": dSize,
		}).Debugln("NewCatalog: FILE")

		if f.FileType == SYMLINK {
			f.TargetPath, err = filepath.EvalSymlinks(f.Path)
			if err != nil {
				break
			}
			continue
		}

		if (dSize + f.DestSize) <= (d.SizeTotal - paddBytes) {
			dSize += f.DestSize
		} else if dSize <= (d.SizeTotal-paddBytes) && f.SourceSize > d.SizeTotal-dSize-paddBytes {
			split = true
			f.SplitStartByte = 0
			f.SplitEndByte = (c.Devices[dNum].SizeTotal - paddBytes) - dSize
			f.DestSize = f.SplitEndByte - f.SplitStartByte
			Log.WithFields(logrus.Fields{
				"dSize": dSize, "f.SplitEndByte": f.SplitEndByte, "f.DestSize": f.DestSize,
			}).Debugln("Splitting file!")

		} else {
			// Out of device space, get the next device
			Log.Debugln("Out of device space!")
			if dNum+1 == len(c.Devices) {
				Log.Error("Total backup size is greater than device pool size!")
				notEnoughSpaceError = true
				c.Devices = append(c.Devices, &Device{
					Name: "overrun", MountPoint: "none", SizeTotal: d.SizeTotal,
				})
			} else {
				dNum += 1
				Log.Debugf("NewCatalog: Using c.Devices[%d]", dNum)
				d = c.Devices[dNum]
				dSize = 0
			}
		}

		Log.WithFields(logrus.Fields{
			"dSize": dSize, "d.SizeTotal": d.SizeTotal, "f.DestSize": f.DestSize,
		}).Debugln("NewCatalog: After size calc")

		// Set the UUID destination path
		var gid string
		gid, err = NewID()
		if err != nil {
			break
		}
		f.DestPath = filepath.Join(d.MountPoint, gid)

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

				// Setup the new file and determine if we need to split again
				fNew.SplitStartByte = lastf.SplitEndByte + 1
				fNew.DestPath = filepath.Join(d.MountPoint, gid)
				fNew.SplitEndByte = fNew.SourceSize
				fNew.DestSize = fNew.SourceSize - fNew.SplitStartByte

				// If the file is still larger than the new device, use all of the available space
				if (dSize + fNew.DestSize) >= (d.SizeTotal - paddBytes) {
					// Use the remaining device space
					fNew.SplitEndByte = fNew.SplitStartByte + (d.SizeTotal - dSize - paddBytes)
					fNew.DestSize = fNew.SplitEndByte - fNew.SplitStartByte
				}
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
				if fNew.SplitEndByte == fNew.SourceSize {
					// The file is accounted for, break the loop
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
					c.Devices = append(c.Devices, &Device{
						Name:       "overrun",
						MountPoint: "none",
						SizeTotal:  fNew.SourceSize,
					})
				}
				lastf = fNew
			}
		} else {
			Log.WithFields(logrus.Fields{"deviceName": d.Name, "fileName": f.Name}).Debugln("File destination")
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
