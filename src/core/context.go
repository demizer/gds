package core

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Context stores the application state
type Context struct {
	BackupPath      string `yaml:"backupPath"`
	OutputStreamNum int
	Files           FileList
	Devices         DeviceList
	Catalog         Catalog

	// Minimum number of bytes that must remain on the device before a file is split across devices
	SplitMinSize uint64
}

// NewContext returns an application context
func NewContext() *Context {
	return &Context{OutputStreamNum: 1}
}

// LoadConfigFromPath loads the application config file from a file path and
// returns an application context on completion.
func LoadConfigFromPath(path string) (*Context, error) {
	conf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := NewContext()
	err = yaml.Unmarshal(conf, c)
	if err != nil {
		return nil, err
	}
	// c.Devices.ParseSizes()
	return c, nil
}

// LoadConfigFromBytes process a byte stream of yaml data and returns an
// application context on success.
func LoadConfigFromBytes(config []byte) (*Context, error) {
	c := NewContext()
	err := yaml.Unmarshal(config, c)
	if err != nil {
		return nil, err
	}
	// c.Devices.ParseSizes()
	return c, nil
}
