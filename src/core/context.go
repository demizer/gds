package core

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

// ContextFileHasNoDevicesError is an error indicating that the configuration yaml data does not contain data for backup
// devices. Backup device specification is an absolute requirement.
type ContextFileHasNoDevicesError int

// Error satisfies the Error interface.
func (e ContextFileHasNoDevicesError) Error() string {
	return fmt.Sprint("YAML data did not contain any devices!")
}

// ContextFileDeviceHasNoSize is returned from ContextFromPath() and indicateds a device was found that does not contain size
// information.
type ContextFileDeviceHasNoSize struct {
	Name string
}

// Error satisfies the Error interface.
func (e ContextFileDeviceHasNoSize) Error() string {
	return fmt.Sprintf("sizeTotal is not defined for device %q", e.Name)
}

// ContextFileDeviceHasNoMountPoint is an error returned by ContextFromPath(). It indicates a device in the configuration
// yaml does not contain mount point information.
type ContextFileDeviceHasNoMountPoint struct {
	Name string
}

// Error satisfies the Error interface.
func (e ContextFileDeviceHasNoMountPoint) Error() string {
	return fmt.Sprintf("mountPoint is not defined for device %q", e.Name)
}

// ContextFileDeviceHasNoUUID is an error returned by ContextFromPath(). It indicates a device in the configuration yaml does
// not contain a UUID.
type ContextFileDeviceHasNoUUID struct {
	Name string
}

// Error satisfies the Error interface.
func (e ContextFileDeviceHasNoUUID) Error() string {
	return fmt.Sprintf("UUID is not defined for device %q", e.Name)
}

// ContextFromPath parses a gds config file from a file path and returns a new context or an error.
func ContextFromPath(path string) (*Context, error) {
	conf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewContextFromYaml(conf)
}

// Context contains the application state
type Context struct {
	BackupPath        string  `json:"backupPath" yaml:"backupPath"`
	OutputStreamNum   uint16  `json:"outputStreams" yaml:"outputStreams"`
	PaddingPercentage float64 `json:"paddingPercentage" yaml:"paddingPercentage"`

	SyncStartDate   time.Time `json:"syncStartDate" yaml:"syncStartDate"`
	LastSyncEndDate time.Time `json:"lastSyncEndDate" yaml:"lastSyncEndDate"`

	FileIndex FileIndex `json:"fileIndex"`

	Devices     DeviceList `json:"devices" yaml:"devices"`
	DevicesUsed int        `json:"devicesUsed"` // Counting start at 1

	SyncProgress    *SyncProgressTracker `json:"-"`
	SyncDeviceMount map[int]chan bool    `json:"-"`

	SyncContextSize uint64 `json:"syncContextSize"`

	Exit bool

	errors chan error // All errors generated in the context will appear here. This chan is buffered.
}

// NewContext returns a context ready to use.
func NewContext(bp string, os uint16, files FileIndex, devices DeviceList, pp float64) (*Context, error) {
	c := &Context{
		BackupPath:        bp,
		OutputStreamNum:   os,
		PaddingPercentage: pp,
		SyncStartDate:     time.Now(),
		Devices:           devices,
		SyncDeviceMount:   make(map[int]chan bool),
		FileIndex:         files,
	}
	if c.PaddingPercentage == 0 {
		c.PaddingPercentage = 1.0
	}
	if c.OutputStreamNum == 0 {
		c.OutputStreamNum = 1
	}
	if !strings.Contains(c.BackupPath, fakeTestPath) {
		if err := c.gatherFiles(); err != nil {
			return nil, err
		}
	}
	for x, _ := range c.Devices {
		if c.Devices[x].PaddingPercentage == 0 {
			// This variable is used when computing padding bytes
			c.Devices[x].PaddingPercentage = c.PaddingPercentage
		}
	}
	if err := c.checkSizes(); err != nil {
		return nil, err
	}
	if err := c.catalog(); err != nil {
		return nil, err
	}
	c.SyncProgress = NewSyncProgressTracker(c.Devices)
	return c, nil
}

// NewContextFromJSON will return a new context from a byte encoded JSON data.
func NewContextFromJSON(b []byte) (*Context, error) {
	c := &Context{
		SyncStartDate:   time.Now(),
		OutputStreamNum: 1,
		SyncDeviceMount: make(map[int]chan bool),
	}
	if err := json.Unmarshal(b, c); err != nil {
		return nil, err
	}
	c.SyncProgress = NewSyncProgressTracker(c.Devices)
	if c.PaddingPercentage == 0 {
		c.PaddingPercentage = 1.0
	}
	for x, _ := range c.Devices {
		if c.Devices[x].PaddingPercentage == 0 {
			// This variable is used when computing padding bytes
			c.Devices[x].PaddingPercentage = c.PaddingPercentage
		}
	}
	c.FileIndex = FileIndex{}
	c.gatherFiles()
	if err := c.catalog(); err != nil {
		return nil, err
	}
	return c, nil
}

// NewContextFromYaml returns a new context parsed from yaml.
func NewContextFromYaml(config []byte) (*Context, error) {
	c := &Context{
		SyncStartDate:   time.Now(),
		OutputStreamNum: 1,
		SyncDeviceMount: make(map[int]chan bool),
	}
	err := yaml.Unmarshal(config, c)
	if err != nil {
		return nil, err
	}
	// Verify device information, but not much else... yet.
	if len(c.Devices) == 0 {
		return nil, new(ContextFileHasNoDevicesError)
	}
	for _, x := range c.Devices {
		if x.SizeTotal == 0 {
			return nil, ContextFileDeviceHasNoSize{x.Name}
		}
		if len(x.MountPoint) == 0 {
			return nil, ContextFileDeviceHasNoMountPoint{x.Name}
		}
		if len(x.UUID) == 0 {
			return nil, ContextFileDeviceHasNoUUID{x.Name}
		}
	}
	c.SyncProgress = NewSyncProgressTracker(c.Devices)
	if c.PaddingPercentage == 0 {
		c.PaddingPercentage = 1.0
	}
	for x, _ := range c.Devices {
		if c.Devices[x].PaddingPercentage == 0 {
			// This variable is used when computing padding bytes
			c.Devices[x].PaddingPercentage = c.PaddingPercentage
		}
	}
	c.FileIndex = FileIndex{}
	c.gatherFiles()
	if err := c.catalog(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Context) checkSizes() error {
	dSize := c.Devices.TotalSizePadded()
	fSize := c.FileIndex.TotalSizeFiles()
	Log.Debugf("checkSizes(): TotalFileSize: %d DeviceSizeWithPadding: %d ", fSize, dSize)
	if fSize > dSize {
		return DevicePoolSizeExceeded{c.FileIndex.TotalSizeFiles(), c.Devices.TotalSize(), c.Devices.TotalSizePadded()}
	}
	return nil
}

// gatherFiles walks the backup paths and loads the file index with file data.
func (c *Context) gatherFiles() error {
	WalkFunc := func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return FileSourceNotReadable{p, fmt.Sprintf("gatherFiles: %s", err.Error())}
		}
		if info.IsDir() && p == c.BackupPath && p[len(p)-1] == '/' {
			return nil
		}
		f := &File{
			Name:    info.Name(),
			Path:    p,
			Size:    uint64(info.Size()),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			AccTime: time.Unix(info.Sys().(*syscall.Stat_t).Atim.Unix()),
			ChgTime: time.Unix(info.Sys().(*syscall.Stat_t).Ctim.Unix()),
			Owner:   int(info.Sys().(*syscall.Stat_t).Uid),
			Group:   int(info.Sys().(*syscall.Stat_t).Gid),
			sha1:    sha1.New(),
		}
		if info.IsDir() {
			f.FileType = DIRECTORY
		} else if info.Mode()&os.ModeSymlink != 0 {
			f.FileType = SYMLINK
		}
		c.FileIndex.Add(f)
		return nil
	}
	return filepath.Walk(c.BackupPath, WalkFunc)
}

// catalogTracker trackes the state of the cataloging process
type catalogTracker struct {
	ctx *Context

	size uint64 // Tracks the size of files stored on the current device. Reset to zero on next device.

	file         *File     // Current file being tracked
	destFile     *DestFile // Destination data for the current file
	destFilePrev *DestFile // The previous destination file

	device       *Device // Current device being tracked
	deviceNumber int
}

func newCatalogTracker(c *Context) *catalogTracker {
	return &catalogTracker{ctx: c, device: c.Devices[0]}
}

func (ct *catalogTracker) nextDevice() error {
	Log.WithFields(logrus.Fields{
		"deviceNum":       ct.deviceNumber,
		"nextDeviceNum":   ct.deviceNumber + 1,
		"numberOfDevices": len(ct.ctx.Devices),
	}).Debugln("nextDevice")
	ct.deviceNumber += 1
	ct.device = ct.ctx.Devices[ct.deviceNumber]
	ct.size = 0
	return nil
}

// splitCheck returns true if the passed file will need to be split based on the byte space remaining for the current device.
func (ct *catalogTracker) splitCheck() bool {
	Log.Debugf("splitCheck: ct.size: %d ct.file.size: %d dev.SizeTotalPadded: %d",
		ct.size, ct.file.Size, ct.device.SizeTotalPadded())
	if (ct.size + ct.file.Size) <= ct.device.SizeTotalPadded() {
		ct.size += ct.file.Size
	} else if ct.size < ct.device.SizeTotalPadded() && ct.file.Size > ct.device.SizeTotalPadded()-ct.size {
		return true
	}
	return false
}

func (ct *catalogTracker) splitEndByteCalc() {
	avail := ct.device.SizeTotalPadded() - ct.size
	remain := ct.file.Size - ct.destFile.StartByte
	Log.Debugf("Remain: %d Avail: %d", remain, avail)
	ct.destFile.EndByte = ct.destFile.StartByte + remain
	if avail > 0 && remain > avail {
		Log.Debugln("Using the remaining device space")
		ct.destFile.EndByte = ct.destFile.StartByte + avail
	}
	ct.destFile.Size = ct.destFile.EndByte - ct.destFile.StartByte
}

// splitFile will split the current file for the current device and loop through remaining files and devices until all
// splitting is done.
func (ct *catalogTracker) splitFile() error {
	// Set the size of the current destination file
	ct.splitEndByteCalc()

	// Increment next device
	ct.file.AddDestFile(ct.destFile)
	ct.destFilePrev = ct.destFile
	if err := ct.nextDevice(); err != nil {
		return err
	}

	// Loop until the file is completely accounted for, across devices if necessary
	ct.debugPrintSplit("Before loop")
	for {
		ct.destFile = NewDestFile(ct.file, ct.device, ct.destFilePrev, ct.destFilePrev)
		if (ct.device.SizeTotalPadded() - ct.size) == 0 {
			if err := ct.nextDevice(); err != nil {
				return err
			}
		}
		// If the file is still larger than the new device, use all of the available space
		if (ct.size + ct.destFile.Size) >= ct.device.SizeTotalPadded() {
			// Use the remaining device space
			ct.debugPrintSplit("Before size calc")
			ct.splitEndByteCalc()
			ct.debugPrintSplit("After size calc")
		} else {
			ct.destFile.Size = ct.destFile.EndByte - ct.destFile.StartByte
		}

		ct.size += ct.destFile.Size
		ct.file.AddDestFile(ct.destFile)

		if ct.destFile.EndByte == ct.file.Size {
			// The file is accounted for, break the loop
			break
		}
		if err := ct.nextDevice(); err != nil {
			return err
		}
		ct.destFilePrev = ct.destFile
	}
	return nil
}

func (ct *catalogTracker) debugPrintSplit(msg string) {
	Log.WithFields(logrus.Fields{
		"ct.size":               ct.size,
		"ct.file.Name":          ct.file.Name,
		"ct.destFile.Size":      ct.destFile.Size,
		"ct.destFile.StartByte": ct.destFile.StartByte,
		"ct.destFile.EndByte":   ct.destFile.EndByte,
		"deviceName":            ct.device.Name,
		"deviceSize":            ct.device.SizeTotal,
		"fileRemaining":         ct.file.Size - ct.destFile.EndByte,
	}).Debugln("Split File:", msg)
}

// catalog determines to which device a file will be saved. Files that won't completely fit on one device will be split
// across devices.
func (c *Context) catalog() error {
	ct := newCatalogTracker(c)

	// Let's light this candle
	for _, file := range ct.ctx.FileIndex {

		Log.Debugf("Inspecting: %s - %d Bytes", file.Name, file.Size)

		// Directories can be ignored, symlinks only need the symlink target set.
		if file.FileType == DIRECTORY {
			continue
		} else if file.FileType == SYMLINK {
			if err := file.SetSymlinkTargetPath(); err != nil {
				c.errors <- err
			}
			continue
		}

		ct.file = file
		if (ct.device.SizeTotalPadded() - ct.size) == 0 {
			ct.nextDevice()
		}
		ct.destFile = NewDestFile(file, ct.device, nil, nil)

		if ct.splitCheck() {
			Log.WithFields(logrus.Fields{
				"deviceSize": ct.device.SizeTotal,
				"ct.size":    ct.size,
				"file":       ct.file.Name,
				"size":       ct.file.Size}).Debugf("Splitting")
			if err := ct.splitFile(); err != nil {
				return err
			}
			continue
		}
		file.AddDestFile(ct.destFile)
	}
	c.DevicesUsed = ct.deviceNumber + 1
	return nil
}
