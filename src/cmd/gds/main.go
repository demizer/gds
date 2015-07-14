package main

import (
	"os"

	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()

	app.Name = "Ghetto Device Storage (gds)"
	app.Version = "0.0.1"
	app.Email = "jeezusjr@gmail.com"
	app.Author = "Jesus Alvarez"
	app.Usage = "Large data backups to dissimilar devices."

	app.Commands = []cli.Command{
		NewRunCommand(),
	}

	app.Run(os.Args)
}
