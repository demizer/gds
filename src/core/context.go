package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Context contains the application state
type Context struct {
	BackupPath      string     `json:"backupPath" yaml:"backupPath"`
	OutputStreamNum int        `json:"outputStreams" yaml:"outputStreams"`
	Files           FileList   `json:"files"`
	Devices         DeviceList `json:"devices" yaml:"devices"`
	Catalog         Catalog    `json:"catalog"`

	// Minimum number of bytes that must remain on the device before a file is split across devices
	SplitMinSize uint64 `yaml:"splitMinSize"`
}

// NewContext returns an app context set to the path from where the backup will be made.
func NewContext(backupPath string) *Context {
	return &Context{
		BackupPath:      backupPath,
		OutputStreamNum: 1,
	}
}

func NewContextFromJSON(b []byte) (*Context, error) {
	var err error
	c := NewContext("")
	err = json.Unmarshal(b, c)
	return c, err
}

// ContextFileHasNoDevicesError is an error indicating that the configuration yaml data does not contain data for backup
// devices. Backup device specification is an absolute requirement.
type ContextFileHasNoDevicesError int

func (e ContextFileHasNoDevicesError) Error() string {
	return fmt.Sprint("YAML data did not contain any devices!")
}

// ContextFromBytes parses a yaml encoded context from a slice of bytes.
func ContextFromBytes(config []byte) (*Context, error) {
	c := &Context{}
	err := yaml.Unmarshal(config, c)
	if err != nil {
		return nil, err
	}
	if len(c.Devices) == 0 {
		return nil, new(ContextFileHasNoDevicesError)
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
