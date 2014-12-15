package main

import (
	"fmt"
	"log"

	"github.com/dustin/go-humanize"
)

type Drive struct {
	Name      string
	UsedSize  uint64
	TotalSize uint64
}

type DriveList []Drive

func (D *DriveList) TotalSize() uint64 {
	var total uint64
	for _, x := range *D {
		total += x.TotalSize
	}
	return total
}

func (d *DriveList) AvailableSpace(drive int, f File) int {
	drv := (*d)[drive]
	if drv.UsedSize+f.Size+PADDING >= drv.TotalSize {
		var s string
		log.Printf("Drive %q is full! (%s of %s) Mount new drive "+
			"and press enter to continue...",
			drv.Name, humanize.IBytes(drv.UsedSize),
			humanize.IBytes(drv.UsedSize))
		fmt.Scanf("%s", &s)
		return drive + 1
	}
	return drive
}
