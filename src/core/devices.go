package core

import (
	"fmt"

	"github.com/dustin/go-humanize"

	log "gopkg.in/inconshreveable/log15.v2"
)

type Device struct {
	Name      string
	UsedSize  uint64
	Size      string
	SizeBytes uint64
}

type DeviceList []Device

func (d *DeviceList) ParseSizes() {
	for x, y := range *d {
		var err error
		(*d)[x].SizeBytes, err = humanize.ParseBytes(y.Size)
		if err != nil {
			log.Crit("Could not parse size!", "err", err.Error())
		}
	}
}

func (d *DeviceList) TotalSize() (uint64, error) {
	var total uint64
	var err error
	for _, x := range *d {
		x.SizeBytes, err = humanize.ParseBytes(x.Size)
		if err != nil {
			return 0, err
		}
		total += x.SizeBytes
	}
	return total, err
}

func (d *DeviceList) AvailableSpace(device int, f File) (int, error) {
	drv := (*d)[device]
	padd, err := humanize.ParseBytes(STATE.Config.Padding)
	if err != nil {
		return 0, err
	}
	if drv.UsedSize+f.Size+padd >= drv.SizeBytes {
		var s string
		log.Info("The device is full! Mount new device and press "+
			"enter to continue...", "currentDevice", drv.Name,
			"used", humanize.IBytes(drv.UsedSize), "totalSize",
			humanize.IBytes(drv.UsedSize))
		fmt.Scanf("%s", &s)
		return device + 1, nil
	}
	return device, nil
}
