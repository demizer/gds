package core

import (
	"time"

	"github.com/Sirupsen/logrus"
)

type bpsRecord struct {
	bps                      *bytesPerSecond
	lastReportedBytesWritten uint64
}

type fileTracker struct {
	io     *IoReaderWriter
	file   *File
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
		sp.Device[x].files = make(chan fileTracker, 10)
		sp.Device[x].Report = make(chan SyncDeviceProgress, 10)
	}
	sp.bps = newBytesPerSecond()
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

	s.bps.addPoint(diffSizeWritn)

	nbps := s.bps.calc()
	if finalReport {
		nbps = s.bps.calcFull()
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
	fbps := newBytesPerSecond()
	// Used to track times from the last report
	lr := time.Now()
	dev := s.devices[index]
	dt := &s.Device[index]
outer:
	for {
		select {
		case bw := <-ft.io.sizeWritn:
			Log.WithFields(logrus.Fields{
				"fileName":                   ft.file.Name,
				"fileDestName":               ft.file.DestPath,
				"fileBytesWritn":             bw,
				"fileTotalBytes":             ft.file.DestSize,
				"elapsedTimeSinceLastReport": time.Since(lr),
				"copyTotalBytesWritn":        ft.io.sizeWritnTotal,
			}).Infoln("Copy report")
			dev.SizeWritn += bw
			dt.bps.addPoint(bw)
			size += bw
			fbps.addPoint(size)
			s.Device[index].Report <- SyncDeviceProgress{
				FileName:             ft.file.Name,
				FilePath:             ft.file.DestPath,
				FileSize:             ft.file.SourceSize,
				FileSizeWritn:        bw,
				FileTotalSizeWritn:   ft.io.sizeWritnTotal,
				FileBytesPerSecond:   fbps.calc(),
				DeviceSizeWritn:      bw,
				DeviceTotalSizeWritn: dev.SizeWritn,
				DeviceBytesPerSecond: dt.bps.calc(),
			}
			if size == ft.file.DestSize {
				Log.WithFields(logrus.Fields{
					"bw":                   bw,
					"destPath":             ft.file.DestPath,
					"destSize":             ft.file.DestSize,
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
			Log.Debugf("No bytes written to %q on device %q in last second.", ft.file.Name, dev.Name)
			dt.bps.addPoint(0)
			fbps.addPoint(0)
			lr = time.Now()
		}
	}

}

// Reports device progress. Should be called every second.
func (s *SyncProgressTracker) deviceCopyReporter(index int) {
	s.Device[index].bps = newBytesPerSecond()
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
