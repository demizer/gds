package core

import (
	"github.com/davecgh/go-spew/spew"
)

var spd = spew.ConfigState{Indent: "\t"} //, DisableMethods: true}

// var STATE *State

// type State struct {
// Config      *Config
// BackupSpace uint64
// }

func Init() {
	// CONFIG_PATH := "/home/demizer/src/gds/data/confidential/config_test.yml"
	// // logs.Debugf("Loading config file from %q\n", CONFIG_PATH)
	// conf, err := LoadConfigFromPath(CONFIG_PATH)
	// if err != nil {
	// log.Crit("Could not open configuration file!", "CONFIG_PATH", CONFIG_PATH)
	// os.Exit(1)
	// }

	// bSpace, err := conf.Devices.TotalSize()
	// if err != nil {
	// log.Crit(err.Error())
	// os.Exit(1)
	// }

	// STATE = &State{
	// Config:      conf,
	// BackupSpace: bSpace,
	// }

	// log.Info("Backup pool stats", "devices", len(STATE.Config.Devices), "total_size", humanize.IBytes(bSpace))

	// log.Info("Gathering a list of files to backup...")
	// files, err := NewFileList(STATE.Config.BackupPath)
	// if err != nil {
	// log.Crit(err.Error())
	// os.Exit(1)
	// }
	// // spd.Dump(files)

	// _, err = json.Marshal(files)
	// if err != nil {
	// log.Crit(err.Error())
	// os.Exit(1)
	// }
	// log.Info("Number of Files", "count", len(*files), "total_size", humanize.IBytes(files.TotalDataSize()))

	// if err := WriteStuff(); err != nil {
	// log.Criticalln(err)
	// os.Exit(1)

}
