package main

import (
	"core"
	"os"

	"github.com/codegangsta/cli"

	"github.com/Sirupsen/logrus"
)

var log = &logrus.Logger{
	Out:       os.Stdout,
	Formatter: new(core.TextFormatter),
	Hooks:     make(logrus.LevelHooks),
	// Level:     logrus.DebugLevel,
	Level: logrus.InfoLevel,
}

func init() {
	core.Log = log
}

func NewSyncCommand() cli.Command {
	return cli.Command{
		Name:  "sync",
		Usage: "Synchronize files to devices",
		Action: func(c *cli.Context) {
			sync(c)
		},
	}
}

func sync(c *cli.Context) {
	log.WithFields(logrus.Fields{"version": 1.1}).Infoln("Ghetto Device Storage")
	c2, err := core.ContextFromPath(c.GlobalString("config"))
	if err != nil {
		log.Fatalf("Error loading config: %s", err.Error())
		os.Exit(1)
	}
	// spd.Dump(c2)
	// os.Exit(1)
	c2.Files, err = core.NewFileList(c2)
	if err != nil {
		log.Fatalf("Error retrieving FileList %s", err.Error())
		os.Exit(1)
	}
	c2.Catalog = core.NewCatalog(c2)
	errs := core.Sync(c2)
	if len(errs) > 0 {
		for _, e := range errs {
			log.Errorf("Sync error: %s", e.Error())
		}
	}
	log.Info("ALL DONE -- Sync complete!")
}
