package main

import (
	"os"
	"time"

	"github.com/codegangsta/cli"
	"github.com/davecgh/go-spew/spew"
)

var spd = spew.ConfigState{Indent: "\t"} //, DisableMethods: true}

func main() {
	app := cli.NewApp()

	app.Name = "Ghetto Device Storage (gds)"
	app.Version = "0.0.1"
	app.Email = "jeezusjr@gmail.com"
	app.Author = "Jesus Alvarez"
	app.Usage = "Large data backups to dissimilar devices."

	t := time.Now()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config,c",
			Value: "$HOME/.config/gds/config.yml",
			Usage: "Load configuration from path",
		},
		cli.StringFlag{
			Name:  "context-path,C",
			Value: "$HOME/.config/gds/context_" + t.Format(time.RFC3339) + ".json",
			Usage: "Change the default path of the context output file path.",
		},
		cli.BoolFlag{
			Name:  "store-context,s",
			Usage: "Store a copy of the context data on every device.",
		},
	}
	app.Commands = []cli.Command{
		NewSyncCommand(),
	}
	app.Run(os.Args)
}
