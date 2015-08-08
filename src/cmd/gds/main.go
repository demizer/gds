package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/codegangsta/cli"
	"github.com/davecgh/go-spew/spew"
)

var spd = spew.ConfigState{Indent: "\t"} //, DisableMethods: true}

var (
	GDS_CONFIG_DIR       = "$HOME/.config/gds"
	GDS_CONTEXT_FILENAME = "context_" + time.Now().Format(time.RFC3339) + ".json"
	GDS_CONFIG_NAME      = "config.yaml"
)

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
			Name:  "config-file,c",
			Value: filepath.Join("$GDS_CONFIG_DIR", GDS_CONFIG_NAME),
			Usage: "Load configuration from path.",
		},
		cli.StringFlag{
			Name:  "context-file,d",
			Value: filepath.Join("$GDS_CONFIG_DIR", GDS_CONTEXT_FILENAME),
			Usage: "the parent directory of sync context files.",
		},
		cli.BoolFlag{
			Name:  "save-context,s",
			Usage: "Save a compressed copy of the sync context data on every device.",
		},
	}
	app.Commands = []cli.Command{
		NewSyncCommand(),
	}
	app.Run(os.Args)
}
