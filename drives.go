package main

import (
	"fmt"

	"github.com/dustin/go-humanize"

	log "gopkg.in/inconshreveable/log15.v2"
)

type Drive struct {
	Name      string
	UsedSize  uint64
	Size      string
	SizeBytes uint64
}

type DriveList []Drive

func (d *DriveList) ParseSizes() {
	for x, y := range *d {
		var err error
		(*d)[x].SizeBytes, err = humanize.ParseBytes(y.Size)
		if err != nil {
			log.Crit("Could not parse size!", "err", err.Error())
		}
	}
}

func (d *DriveList) TotalSize() (uint64, error) {
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

func (d *DriveList) AvailableSpace(drive int, f File) (int, error) {
	drv := (*d)[drive]
	padd, err := humanize.ParseBytes(STATE.Config.Padding)
	if err != nil {
		return 0, err
	}
	if drv.UsedSize+f.Size+padd >= drv.SizeBytes {
		var s string
		log.Info("Drive is full! Mount new drive and press "+
			"enter to continue...", "currentDrive", drv.Name,
			"used", humanize.IBytes(drv.UsedSize), "totalSize",
			humanize.IBytes(drv.UsedSize))
		fmt.Scanf("%s", &s)
		return drive + 1, nil
	}
	return drive, nil
}
