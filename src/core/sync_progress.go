package core

import (
	"time"

	"github.com/Sirupsen/logrus"
)

type bpsRecord struct {
	bps                      *BytesPerSecond
	lastReportedBytesWritten uint64
}

type fileTracker struct {
	io     *IoReaderWriter
	f      *File
	df     *DestFile
	device *Device
	closed bool
	done   chan bool
	bpsRecord
}

type deviceTracker struct {
	files chan fileTracker
	bpsRecord
	Report chan SyncDeviceProgress
}

// SyncProgress details information of the overall sync progress.
type SyncProgress struct {
	SizeWritn      uint64
	SizeTotal      uint64
	BytesPerSecond uint64
	ETA            time.Time
}

// SyncDeviceProgress contains information for a file copy in progress.
type SyncDeviceProgress struct {
	FileName string
	FilePath string
	FileSize uint64

	FileSizeWritn      uint64 // Number of bytes written since last report
	FileTotalSizeWritn uint64 // Total number of bytes written to dest file
	FileBytesPerSecond uint64

	DeviceSizeWritn      uint64 // Number of bytes written since last report
	DeviceTotalSizeWritn uint64 // Total number of bytes written to dest device
	DeviceBytesPerSecond uint64
}

// SyncProgressTracker is used to report copy progress.
type SyncProgressTracker struct {
	bpsRecord
	Report chan SyncProgress

	devices DeviceList
	Device  []deviceTracker
}

// NewSyncProgressTracker returns a fresh SyncProgressTracker object.
func NewSyncProgressTracker(devices DeviceList) *SyncProgressTracker {
	sp := &SyncProgressTracker{
		devices: devices,
		Report:  make(chan SyncProgress, len(devices)),
	}
	for x := 0; x < len(devices); x++ {
		sp.Device = append(sp.Device, deviceTracker{})
		sp.Device[x].files = make(chan fileTracker)
		sp.Device[x].Report = make(chan SyncDeviceProgress)
	}
	sp.bps = NewBytesPerSecond()
	return sp
}

// Reports overall progress. Should be called once a second.
func (s *SyncProgressTracker) report(finalReport bool) {
	totalSyncBytesWritten := s.devices.TotalSizeWritten()
	if totalSyncBytesWritten == 0 {
		return
	}
	diffSizeWritn := totalSyncBytesWritten - s.lastReportedBytesWritten

	Log.WithFields(logrus.Fields{
		"totalSyncBytesWritn": totalSyncBytesWritten,
		"lastBytesWritn":      s.lastReportedBytesWritten,
		"diff":                diffSizeWritn,
	}).Infoln("Overall progress report")

	s.bps.AddPoint(diffSizeWritn)

	nbps := s.bps.Calc()
	if finalReport {
		nbps = s.bps.CalcFull()
	}
	s.Report <- SyncProgress{
		SizeWritn:      totalSyncBytesWritten,
		BytesPerSecond: nbps,
	}

	s.lastReportedBytesWritten = totalSyncBytesWritten
}

func (s *SyncProgressTracker) fileCopyReporter(index int, ft fileTracker) {
	// Tracks total file size reported to the fileTracker
	var size uint64
	// File bps calculation
	fbps := NewBytesPerSecond()
	// Used to track times from the last report
	lr := time.Now()
	dev := s.devices[index]
	dt := &s.Device[index]
outer:
	for {
		select {
		case bw := <-ft.io.sizeWritn:
			Log.WithFields(logrus.Fields{
				"fileName":                   ft.f.Name,
				"fileDestPath":               ft.f.Path,
				"fileBytesWritn":             bw,
				"fileTotalBytes":             ft.f.Size,
				"elapsedTimeSinceLastReport": time.Since(lr),
				"copyTotalBytesWritn":        ft.io.sizeWritnTotal,
			}).Infoln("Copy report")
			dev.SizeWritn += bw
			dt.bps.AddPoint(bw)
			size += bw
			fbps.AddPoint(size)
			s.Device[index].Report <- SyncDeviceProgress{
				FileName:             ft.f.Name,
				FilePath:             ft.df.Path,
				FileSize:             ft.df.Size,
				FileSizeWritn:        bw,
				FileTotalSizeWritn:   ft.io.sizeWritnTotal,
				FileBytesPerSecond:   fbps.Calc(),
				DeviceSizeWritn:      bw,
				DeviceTotalSizeWritn: dev.SizeWritn,
				DeviceBytesPerSecond: dt.bps.Calc(),
			}
			if size == ft.df.Size {
				Log.WithFields(logrus.Fields{
					"bw":                   bw,
					"destPath":             ft.f.Path,
					"destSize":             ft.f.Size,
					"ft.io.sizeWritnTotal": ft.io.sizeWritnTotal,
				}).Print("Copy complete")
				// Test the sync goroutine that file is accounted for
				ft.done <- true
				break outer
			}
			lr = time.Now()
		case <-time.After(time.Second):
			if ft.closed {
				Log.Debugln("Tracker loop has been closed. Exiting.")
				break outer
			}
			Log.Debugf("No bytes written to %q on device %q in last second.", ft.f.Name, dev.Name)
			dt.bps.AddPoint(0)
			fbps.AddPoint(0)
			lr = time.Now()
		}
	}
}

// Reports device progress. Should be called every second.
func (s *SyncProgressTracker) deviceCopyReporter(index int) {
	s.Device[index].bps = NewBytesPerSecond()
	for {
		// Report the progress for each file
		if ft, ok := <-s.Device[index].files; ok {
			s.fileCopyReporter(index, ft)
		} else {
			Log.Debugln("deviceCopyReporter(): Breaking main reporter loop!")
			break
		}
	}
	Log.WithFields(logrus.Fields{
		"index": index, "device.SizeWritn": s.devices[index].SizeWritn,
	}).Debugf("DEVICE COPY DONE")
}
