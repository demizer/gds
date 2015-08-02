package core

// Device represents a single mountable storage device.
type Device struct {
	Name       string
	MountPoint string `yaml:"mountPoint"`
	Size       uint64
	UsedSize   uint64
}

// DeviceList is a type for a list of devices.
type DeviceList []Device

// DevicePoolSize returns the total size in bytes of the device pool.
func (d *DeviceList) DevicePoolSize() uint64 {
	var total uint64
	for _, x := range *d {
		total += x.Size
	}
	return total
}

// DeviceByName returns a pointer to the object of the named device.
func (d *DeviceList) DeviceByName(name string) *Device {
	for x, y := range *d {
		if y.Name == name {
			return &(*d)[x]
		}
	}
	return nil
}
