package main

import (
	"core"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/codegangsta/cli"

	"github.com/Sirupsen/logrus"
)

func NewSyncCommand() cli.Command {
	return cli.Command{
		Name:  "sync",
		Usage: "Synchronize files to devices",
		Action: func(c *cli.Context) {
			if len(os.Args) == 1 {
				cli.ShowAppHelp(c)
				log.Formatter.(*core.TextFormatter).DisableTimestamp = true
				log.Fatalf("No arguments specified!")
			}
			err := checkEnvVariables(c)
			if err != nil {
				log.Fatalf("Could not set environment variables: %s", err)
				os.Exit(1)
			}
			if !c.GlobalBool("no-file-log") {
				lp := cleanPath(c.GlobalString("log"))
				lf, err := os.Create(lp)
				if err != nil {
					log.Fatalf("Could not create log file: %s", err)
					os.Exit(1)
				}
				// log.Out = io.MultiWriter(os.Stdout, lf)
				log.Out = io.MultiWriter(os.Stdout, lf)
			}
			lvl, err := logrus.ParseLevel(c.GlobalString("log-level"))
			if err != nil {
				cli.ShowAppHelp(c)
				log.Formatter.(*core.TextFormatter).DisableTimestamp = true
				log.Fatalf("Error parsing log level: %s", err)
			}
			log.Level = lvl
			sync(c)
		},
	}
}

func sync(c *cli.Context) {
	log.WithFields(logrus.Fields{"version": 0.2}).Infoln("Ghetto Device Storage")

	cPath, err := getConfigFile(c.GlobalString("config"))
	if err != nil {
		log.Fatal(err)
	}
	log.WithFields(logrus.Fields{
		"path": cPath,
	}).Info("Using configuration file")

	cf, err := getContextFile(c.GlobalString("context"))
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

	c2.Catalog, err = core.NewCatalog(c2)
	if err != nil {
		log.Fatal(err)
	}
	errs := core.Sync(c2, c.GlobalBool("no-dev-context"))
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
