package core

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"github.com/demizer/go-humanize"

	log "gopkg.in/inconshreveable/log15.v2"
)

type Config struct {
	BackupPath      string `yaml:"backupPath"`
	DestinationPath string `yaml:"destinationPath"`
	Padding         string
	PaddingBytes    uint64
	Devices         DeviceList
}

func NewConfig() *Config {
	return &Config{Devices: make([]Device, 0)}
}

func (c *Config) parseSizes() {
	c.Devices.ParseSizes()
	var err error
	c.PaddingBytes, err = humanize.ParseBytes(c.Padding)
	if err != nil {
		log.Crit("Could not parse padding size", "err", err.Error())
	}
}

func LoadConfigFromPath(path string) (*Config, error) {
	conf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := NewConfig()
	err = yaml.Unmarshal(conf, c)
	if err != nil {
		return nil, err
	}
	c.parseSizes()
	return c, nil
}

func LoadConfigFromBytes(config []byte) (*Config, error) {
	c := NewConfig()
	err := yaml.Unmarshal(config, c)
	if err != nil {
		return nil, err
	}
	c.parseSizes()
	return c, nil
}
