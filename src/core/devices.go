package core

// Device represents a single mountable storage device.
type Device struct {
	Name       string
	MountPoint string `yaml:"mountPoint"`
	SizeWritn  uint64 `yaml:"sizeWritn"`
	SizeTotal  uint64 `yaml:"sizeTotal"`
	UUID       string
}

// DeviceList is a type for a list of devices.
type DeviceList []Device

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

// DeviceNotFoundError is given when a device name is not found in the device list.
type DeviceNotFoundError int

func (d DeviceNotFoundError) Error() string {
	return "Device not found"
}

// DeviceByName returns a pointer to the object of the named device. Returns DeviceNotFoundError if the device is not in the
// list.
func (d *DeviceList) DeviceByName(name string) (*Device, error) {
	for x, y := range *d {
		if y.Name == name {
			return &(*d)[x], nil
		}
	}
	return nil, new(DeviceNotFoundError)
}
