package main

import (
	"core"

	"github.com/codegangsta/cli"

	log "gopkg.in/inconshreveable/log15.v2"
)

func NewRunCommand() cli.Command {
	return cli.Command{
		Name:  "run",
		Usage: "Run the application server",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "port",
				Value: "9999",
				Usage: "Server listen port",
			},
		},
		Action: func(c *cli.Context) {
			run(c)
		},
	}
}

func run(c *cli.Context) {
	log.Info("Ghetto Device Storage (gds) 0.0.1")
	core.Init()
}
