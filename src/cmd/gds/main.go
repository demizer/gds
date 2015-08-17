package main

import (
	"conui"
	"core"
	"fmt"
	"io"
	"logfmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/davecgh/go-spew/spew"
	"github.com/nsf/termbox-go"
)

var spd = spew.ConfigState{Indent: "\t"} //, DisableMethods: true}

var (
	GDS_LOG_FD           *os.File
	GDS_CONFIG_DIR       = "$HOME/.config/gds"
	GDS_CONTEXT_FILENAME = "context_" + time.Now().Format(time.RFC3339) + ".json"
	GDS_LOG_FILENAME     = "log_" + time.Now().Format(time.RFC3339) + ".log"
	GDS_CONFIG_NAME      = "config.yaml"
)

var log = &logrus.Logger{
	Out:       os.Stdout,
	Formatter: new(logfmt.TextFormatter),
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.InfoLevel,
}

func init() {
	core.Log = log
	conui.Log = log
}

// fatal exit type. Passed as an argument to panic().
type fatal struct {
	err interface{}
}

func (f fatal) Error() (s string) {
	switch f.err.(type) {
	case error:
		s = f.err.(error).Error()
	case string:
		s = f.err.(string)
	}
	return
}

type fatalShowHelp struct {
	err interface{}
}

func (f fatalShowHelp) Error() (s string) {
	switch v := f.err.(type) {
	case error:
		s = v.Error()
	case string:
		s = v
	}
	return
}

// cleanupAtExit performs some cleanup operations before exiting.
func cleanupAtExit(c *cli.App) {
	if termbox.IsInit {
		conui.Close()
	}
	log.Formatter.(*logfmt.TextFormatter).DisableTimestamp = true
	if err := recover(); err != nil {
		logOutReset := func() {
			log.Out = os.Stdout
			if GDS_LOG_FD != nil {
				log.Out = io.MultiWriter(os.Stdout, GDS_LOG_FD)
			}
		}
		logOutReset()
		stack := make([]byte, 4096)
		size := runtime.Stack(stack, true)
		var v string
		switch err.(type) {
		case fatal:
			v = err.(fatal).Error()
		case fatalShowHelp:
			cli.HelpPrinter(os.Stdout, cli.AppHelpTemplate, c)
			log.Fatal(err)
		default:
			v = fmt.Sprint(err)
		}
		log.Errorf("Unexpected failure! See %q for details...", GDS_LOG_FD.Name())
		GDS_LOG_FD.WriteString(fmt.Sprintf("\nFATAL ERROR: %s\n\n%s\n", v, string(stack[:size])))
	}
	// panic("SHOW ME THE STACKS!")
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
	defer cleanupAtExit(app)
	if len(os.Args) == 1 {
		panic(fatalShowHelp{"No arguments specified!"})
	}
	app.Run(os.Args)
}
