package core

import (
	"fmt"
	"github.com/demizer/go-humanize"
)

// DeviceNotFoundError is given when a device name is not found in the device list.
type DeviceNotFoundError int

func (d DeviceNotFoundError) Error() string {
	return "Device not found"
}

// DevicePoolSizeExceeded is an error given when the backup size exceeds the device pool storage size.
type DevicePoolSizeExceeded struct {
	TotalIndexSize            uint64
	TotalDevicePoolSize       uint64
	TotalPaddedDevicePoolSize uint64
}

// Error implements the Error interface.
func (e DevicePoolSizeExceeded) Error() string {
	return fmt.Sprintf("Inadequate device pool space! TotalIndexSize: %d (%s) TotalPaddedDevicePoolSize: %d (%s)",
		e.TotalIndexSize, humanize.IBytes(e.TotalIndexSize), e.TotalPaddedDevicePoolSize,
		humanize.IBytes(e.TotalPaddedDevicePoolSize))
}

// Device represents a single mountable storage device.
type Device struct {
	Name              string
	MountPoint        string  `yaml:"mountPoint"`
	SizeWritn         uint64  `yaml:"sizeWritn"`
	SizeTotal         uint64  `yaml:"sizeTotal"`
	PaddingPercentage float64 `yaml:"paddingPercentage"`
	UUID              string
	files             []*DestFile
}

// SizeTotalPadded returns the device total size with the defined percentage of padding bytes subtracted.
func (d *Device) SizeTotalPadded() uint64 {
	return d.SizeTotal - uint64(float64(d.SizeTotal)*(d.PaddingPercentage/100))
}

// SizePaddingBytes returns the number of bytes used for padding.
func (d *Device) SizePaddingBytes() uint64 {
	return uint64(float64(d.SizeTotal) * (d.PaddingPercentage / 100))
}

// DeviceList is a type for a list of devices.
type DeviceList []*Device

// Add a device to the device list.
func (d *DeviceList) Add(dev *Device) {
	*d = append(*d, dev)
}

// TotalSize returns the total size in bytes of the device pool.
func (d *DeviceList) TotalSize() uint64 {
	var total uint64
	for _, x := range *d {
		if x.Name == "overrun" {
			// NewCatalog() creates devices named "overrun", when the pool size has been exceeded when splitting
			// a file across devices. It is necessary to create a new device so that the actual data size and
			// device pool size can be calculated and reported to the user.
			continue
		}
		total += x.SizeTotal
	}
	return total
}

// TotalSizePadded returns the total size in bytes, subtracting a number of bytes defined by the padding percentage specified
// in the configuration yaml. Default padding bytes subtracted is 1 percent of device size.
func (d *DeviceList) TotalSizePadded() uint64 {
	var total uint64
	for _, x := range *d {
		paddBytes := uint64(float64(x.SizeTotal) * (x.PaddingPercentage / 100))
		total += x.SizeTotal - paddBytes
	}
	return total
}

// TotalSizeWritten returns the total bytes written to the device pool.
func (d *DeviceList) TotalSizeWritten() uint64 {
	var total uint64
	for _, x := range *d {
		if x.Name == "overrun" {
			// See comment in d.TotalSize()
			continue
		}
		total += x.SizeWritn
	}
	return total
}

// DeviceByName returns a pointer to the object of the named device. Returns DeviceNotFoundError if the device is not in the
// list.
func (d *DeviceList) DeviceByName(name string) (*Device, error) {
	for x, y := range *d {
		if y.Name == name {
			return (*d)[x], nil
		}
	}
	return nil, new(DeviceNotFoundError)
}
