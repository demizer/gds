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
	BackupPath        string     `json:"backupPath" yaml:"backupPath"`
	OutputStreamNum   int        `json:"outputStreams" yaml:"outputStreams"`
	SyncStartDate     time.Time  `json:"syncStartDate" yaml:"syncStartDate"`
	LastSyncEndDate   time.Time  `json:"lastSyncEndDate" yaml:"lastSyncEndDate"`
	PaddingPercentage float64    `json:"paddingPercentage" yaml:"paddingPercentage"`
	Files             FileList   `json:"files"`
	Devices           DeviceList `json:"devices" yaml:"devices"`
	Catalog           Catalog    `json:"catalog"`

	// Minimum number of bytes that must remain on the device before a file is split across devices
	SplitMinSize uint64 `json:"splitMinSize" yaml:"splitMinSize"`

	// Progress communication channels
	SyncProgress       chan SyncProgress               `json:"-"`
	SyncDeviceProgress map[int]chan SyncDeviceProgress `json:"-"`
	SyncDeviceMount    map[int]chan bool               `json:"-"`

	Exit bool
}

// NewContext returns a new core Context ready to use.
func NewContext() *Context {
	return &Context{
		SyncStartDate:      time.Now(),
		OutputStreamNum:    1,
		PaddingPercentage:  1.0,
		SyncProgress:       make(chan SyncProgress),
		SyncDeviceProgress: make(map[int]chan SyncDeviceProgress),
		SyncDeviceMount:    make(map[int]chan bool),
	}
}

func NewContextFromJSON(b []byte) (*Context, error) {
	var err error
	c := NewContext()
	err = json.Unmarshal(b, c)
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

// ContextFromBytes parses a yaml encoded context from a slice of bytes.
func ContextFromBytes(config []byte) (*Context, error) {
	c := NewContext()
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
	return c, nil
}

// ContextFromPath parses a gds config file from a file path and returns a new context or an error.
func ContextFromPath(path string) (*Context, error) {
	conf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ContextFromBytes(conf)
}
