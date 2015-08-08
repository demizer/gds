package main

import (
	"core"
	"encoding/json"
	"io/ioutil"
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
			err := checkEnvVariables(c)
			if err != nil {
				log.Fatalf("Could not set environment variables: %s", err)
				os.Exit(1)
			}
			sync(c)
		},
	}
}

func sync(c *cli.Context) {
	log.WithFields(logrus.Fields{"version": 0.2}).Infoln("Ghetto Device Storage")

	cPath, err := getConfigFile(c.GlobalString("config-file"))
	if err != nil {
		log.Fatal(err)
	}
	log.WithFields(logrus.Fields{
		"path": cPath,
	}).Info("Using configuration file")

	cf, err := getContextFile(c.GlobalString("context-file"))
	if err != nil {
		log.Fatalf("Could not create context JSON output file: %s", err.Error())
		os.Exit(1)
	}

	c2, err := core.ContextFromPath(cPath)
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

	j, err := json.Marshal(c2)
	if err == nil {
		err = ioutil.WriteFile(cf, j, 0644)
	}
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Info("ALL DONE -- Sync complete!")
}
