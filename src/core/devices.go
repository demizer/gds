package core

// Device represents a single mountable storage device.
type Device struct {
	Name       string
	UsedSize   uint64
	Size       string
	SizeBytes  uint64
	MountPoint string
}

// DeviceList is a type for a list of devices.
type DeviceList []Device

// DevicePoolSize returns the total size in bytes of the device pool.
func (d *DeviceList) DevicePoolSize() uint64 {
	var total uint64
	for _, x := range *d {
		total += x.SizeBytes
	}
	return total
}

// GetDeviceByName returns a pointer to the object of the named device.
func (d *DeviceList) GetDeviceByName(name string) *Device {
	for x, y := range *d {
		if y.Name == name {
			return &(*d)[x]
		}
	}
	return nil
}

// SetMountPointByName sets the mount point of a device using the name of the device.
func (d *DeviceList) SetMountPointByName(name string, mountPoint string) {
	for x, y := range *d {
		if y.Name == name {
			(*d)[x].MountPoint = mountPoint
			return
		}
	}
}

// func (d *DeviceList) Count() int {
// return len(*d)
// }

// func (d *DeviceList) ParseSizes() {
// for x, y := range *d {
// var err error
// (*d)[x].SizeBytes, err = humanize.ParseBytes(y.Size)
// if err != nil {
// log.Crit("Could not parse size!", "err", err.Error())
// }
// }
// }

// func (d *DeviceList) AvailableSpace(device int, f File) (int, error) {
// drv := (*d)[device]
// var padd uint64 = 1048576
// if drv.UsedSize+f.Size+padd >= drv.SizeBytes {
// var s string
// log.Info("The device is full! Mount new device and press "+
// "enter to continue...", "currentDevice", drv.Name,
// "used", humanize.IBytes(drv.UsedSize), "DevicePoolSize",
// humanize.IBytes(drv.UsedSize))
// fmt.Scanf("%s", &s)
// return device + 1, nil
// }
// return device, nil
// }
