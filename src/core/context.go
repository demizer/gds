package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

// Context contains the application state
type Context struct {
	BackupPath        string  `json:"backupPath" yaml:"backupPath"`
	OutputStreamNum   uint16  `json:"outputStreams" yaml:"outputStreams"`
	PaddingPercentage float64 `json:"paddingPercentage" yaml:"paddingPercentage"`

	SyncStartDate   time.Time `json:"syncStartDate" yaml:"syncStartDate"`
	LastSyncEndDate time.Time `json:"lastSyncEndDate" yaml:"lastSyncEndDate"`

	Files   FileList   `json:"files"`
	Devices DeviceList `json:"devices" yaml:"devices"`
	Catalog Catalog    `json:"catalog"`

	SyncProgress    *SyncProgressTracker `json:"-"`
	SyncDeviceMount map[int]chan bool    `json:"-"`

	SyncContextSize uint64 `json:"syncContextSize"`

	Exit bool
}

// NewContext returns a context ready to use.
func NewContext(backupPath string, outputStreams uint16, files FileList, devices DeviceList, paddingPercentage float64) *Context {
	c := &Context{
		BackupPath:        backupPath,
		OutputStreamNum:   outputStreams,
		PaddingPercentage: paddingPercentage,
		SyncStartDate:     time.Now(),
		Devices:           devices,
		Files:             files,
		SyncDeviceMount:   make(map[int]chan bool),
	}
	if c.PaddingPercentage == 0 {
		c.PaddingPercentage = 1.0
	}
	if c.OutputStreamNum == 0 {
		c.OutputStreamNum = 1
	}
	c.SyncProgress = NewSyncProgressTracker(c.Devices)
	for x, _ := range c.Devices {
		if c.Devices[x].PaddingPercentage == 0 {
			// This variable is used when computing padding bytes
			c.Devices[x].PaddingPercentage = c.PaddingPercentage
		}
	}
	return c
}

func NewContextFromJSON(b []byte) (*Context, error) {
	var err error
	c := &Context{
		SyncStartDate:   time.Now(),
		OutputStreamNum: 1,
		SyncDeviceMount: make(map[int]chan bool),
	}
	err = json.Unmarshal(b, c)
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
	return c, err
}

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
	return c, nil
}

// ContextFromPath parses a gds config file from a file path and returns a new context or an error.
func ContextFromPath(path string) (*Context, error) {
	conf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewContextFromYaml(conf)
}
