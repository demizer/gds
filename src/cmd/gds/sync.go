package main

import (
	"core"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
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
				log.Out = io.MultiWriter(lf)
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

// loadInitialState prepares the applicaton for usage
func loadInitialState(c *cli.Context) *core.Context {
	cPath, err := getConfigFile(c.GlobalString("config"))
	if err != nil {
		log.Fatal(err)
	}
	log.WithFields(logrus.Fields{
		"path": cPath,
	}).Info("Using configuration file")

	c2, err := core.ContextFromPath(cPath)
	if err != nil {
		log.Fatalf("Error loading config: %s", err.Error())
		os.Exit(1)
	}

	c2.Files, err = core.NewFileList(c2)
	if err != nil {
		log.Fatalf("Error retrieving FileList %s", err.Error())
		os.Exit(1)
	}

	c2.Catalog, err = core.NewCatalog(c2)
	if err != nil {
		log.Fatal(err)
	}

	return c2
	// spd.Dump(c2)
	// os.Exit(1)
}

func dumpContextToFile(c *cli.Context, c2 *core.Context) {
	cf, err := getContextFile(c.GlobalString("context"))
	if err != nil {
		log.Fatalf("Could not create context JSON output file: %s", err.Error())
		os.Exit(1)
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
