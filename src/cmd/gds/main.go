package main

import (
	"conui"
	"core"
	"os"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/davecgh/go-spew/spew"
	"github.com/gizak/termui"
)

var spd = spew.ConfigState{Indent: "\t"} //, DisableMethods: true}

var (
	GDS_CONFIG_DIR       = "$HOME/.config/gds"
	GDS_CONTEXT_FILENAME = "context_" + time.Now().Format(time.RFC3339) + ".json"
	GDS_LOG_FILENAME     = "log_" + time.Now().Format(time.RFC3339) + ".log"
	GDS_CONFIG_NAME      = "config.yaml"
)

var log = &logrus.Logger{
	Out:       os.Stdout,
	Formatter: &core.TextFormatter{},
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.InfoLevel,
}

func init() {
	core.Log = log
	conui.Init()
}

// handleFatal closes the termui sessions before dumping the panic info to stdout
func handleFatal() {
	if err := recover(); err != nil {
		termui.Close()
		log.Out = os.Stdout
		panic(err)
	}
}

func main() {
	app := cli.NewApp()

	app.Name = "Ghetto Device Storage (gds)"
	app.Version = "0.0.1"
	app.Email = "jeezusjr@gmail.com"
	app.Author = "Jesus Alvarez"
	app.Usage = "Large data backups to dissimilar devices."

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config-dir,C",
			Value: GDS_CONFIG_DIR,
			Usage: "Change the default configuration directory.",
		},
		cli.StringFlag{
			Name:  "config,c",
			Value: filepath.Join("$GDS_CONFIG_DIR", GDS_CONFIG_NAME),
			Usage: "Load configuration from path.",
		},
		cli.StringFlag{
			Name:  "context,d",
			Value: filepath.Join("$GDS_CONFIG_DIR", GDS_CONTEXT_FILENAME),
			Usage: "the parent directory of sync context files.",
		},
		cli.BoolFlag{
			Name:  "no-dev-context,n",
			Usage: "Do not save a copy of the sync context to the last device.",
		},
		cli.BoolFlag{
			Name:  "no-file-log,x",
			Usage: "Disable file logging.",
		},
		cli.StringFlag{
			Name:  "log,l",
			Value: filepath.Join("$GDS_CONFIG_DIR", GDS_LOG_FILENAME),
			Usage: "Save output log to file.",
		},
		cli.StringFlag{
			Name:  "log-level,L",
			Value: "info",
			Usage: "The level of log output. Levels: debug, info, warn, error, fatal, panic",
		},
	}
	app.Commands = []cli.Command{
		NewSyncCommand(),
	}
	// If a panic occurrs while termui session is active, the panic output is unreadable.
	defer handleFatal()
	app.Run(os.Args)
}
