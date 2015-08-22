package main

import (
	"conui"
	"core"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/gizak/termui"
	"github.com/nsf/termbox-go"
)

func NewSyncCommand() cli.Command {
	return cli.Command{
		Name:  "sync",
		Usage: "Synchronize files to devices",
		Action: func(c *cli.Context) {
			err := checkEnvVariables(c)
			if err != nil {
				panic(fatal{fmt.Sprintf("Could not set environment variables: %s", err)})
			}
			if !c.GlobalBool("no-file-log") {
				lp := cleanPath(c.GlobalString("log"))
				var err error
				GDS_LOG_FD, err = os.Create(lp)
				if err != nil {
					panic(fatal{fmt.Sprintf("Could not create log file: %s", err)})
				}
				log.Out = GDS_LOG_FD
			}
			lvl, err := logrus.ParseLevel(c.GlobalString("log-level"))
			if err != nil {
				panic(fatalShowHelp{fmt.Sprintf("Error parsing log level: %s", err)})
			}
			log.Level = lvl
			syncStart(c)
		},
	}
}

// loadInitialState prepares the applicaton for usage
func loadInitialState(c *cli.Context) *core.Context {
	cPath, err := getConfigFile(c.GlobalString("config"))
	if err != nil {
		panic(fatal{err})
	}
	log.WithFields(logrus.Fields{
		"path": cPath,
	}).Info("Using configuration file")

	c2, err := core.ContextFromPath(cPath)
	if err != nil {
		panic(fatal{fmt.Sprintf("Error loading config: %s", err.Error())})
	}

	c2.Files, err = core.NewFileList(c2)
	if err != nil {
		panic(fatal{fmt.Sprintf("Error retrieving FileList %s", err.Error())})
	}

	c2.Catalog, err = core.NewCatalog(c2)
	if err != nil {
		panic(fatal{err})
	}

	return c2
}

func dumpContextToFile(c *cli.Context, c2 *core.Context) {
	cf, err := getContextFile(c.GlobalString("context"))
	if err != nil {
		panic(fatal{fmt.Sprintf("Could not create context JSON output file: %s", err.Error())})
	}
	j, err := json.Marshal(c2)
	if err == nil {
		err = ioutil.WriteFile(cf, j, 0644)
	}
	if err != nil {
		panic(fatal{fmt.Sprintf("Could not marshal JSON to file: %s", err.Error())})
	}
}

// BuildConsole creates the UI widgets First is the main progress guage for the overall progress Widgets are then created for
// each of the devices, but are hidden initially.
func BuildConsole(c *core.Context) {
	var rows []termui.GridBufferer
	visible := c.OutputStreamNum
	for x, y := range c.Devices {
		conui.Widgets[x] = conui.NewDevicePanel(y.Name, y.SizeTotal)
		if visible > 0 {
			conui.Widgets[x].(*conui.DevicePanel).Visible = true
			if x == 0 {
				conui.Widgets[x].(*conui.DevicePanel).Selected = true
			}
			visible--
		}
	}
	conui.Widgets[len(c.Devices)] = conui.NewProgressGauge(c.Devices.DevicePoolSize())
	rows = append(rows, conui.Widgets[len(c.Devices)])
	for x, _ := range c.Devices {
		rows = append(rows, conui.Widgets[x].(*conui.DevicePanel))
	}
	// log.Debugln(spd.Sdump(rows))
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, rows...),
		),
	)
	termui.Body.Align()
}

func eventListener(c *core.Context) {
	defer cleanupAtExit()
	for {
		select {
		case e := <-conui.Event:
			if e.Type == termui.EventKey && e.Ch == 'j' {
				conui.Widgets.SelectNext()
			}
			if e.Type == termui.EventKey && e.Ch == 'k' {
				conui.Widgets.SelectPrevious()
			}
			if e.Type == termui.EventKey && e.Key == termui.KeyEnter {
				conui.Widgets.Selected().Prompt = conui.Prompt{}
			}
			if e.Type == termui.EventKey && e.Ch == 'q' {
				conui.Close()
				os.Exit(0)
			}
			if e.Type == termui.EventResize {
				termui.Body.Width = termui.TermWidth()
				termui.Body.Align()
				go func() { conui.Redraw <- true }()
			}
		case <-conui.Redraw:
			termui.Render(termui.Body)
		}
	}
}

func update(c *core.Context) {
	for x := 0; x < len(c.Devices); x++ {
		c.SyncDeviceMount[x] = make(chan bool)
		go func(index int) {
			ns := time.Now()
			log.Debugln("Waiting for receive on SyncDeviceMount")
			<-c.SyncDeviceMount[index]
			log.Debugf("Receive from SyncDeviceMount after wait of %s", time.Since(ns))
			wg := conui.Widgets.MountPromptByIndex(index)
			d, err := c.Devices.DeviceByName(wg.Border.Label)
			if err != nil {
				log.Error(err)
			}
			err = ensureDeviceIsMounted(*d)
			if err != nil {
				log.Error(err)
			}
			c.SyncDeviceMount[index] <- true
		}(x)
		c.SyncProgress[x] = make(chan core.SyncProgress, 100)
		c.SyncFileProgress[x] = make(chan core.SyncFileProgress, 100)
		go func(index int) {
			// defer cleanupAtExit()
			for {
				select {
				case <-c.SyncProgress[index]:
				case <-c.SyncFileProgress[index]:
				}
			}
		}(x)
	}
	go func() {
		for {
			if !termbox.IsInit {
				break
			}
			conui.Redraw <- true
			time.Sleep(time.Second / 5)
		}
	}()
}

func syncStart(c *cli.Context) {
	defer cleanupAtExit()
	log.WithFields(logrus.Fields{
		"version": 0.2,
		"date":    time.Now().Format(time.RFC3339),
	}).Infoln("Ghetto Device Storage")
	c2 := loadInitialState(c)

	// CONSOLE UI FROM THE 1980s
	conui.Init()
	BuildConsole(c2)
	go eventListener(c2)
	update(c2)

	// Sync the things
	errs := core.Sync(c2, c.GlobalBool("no-dev-context"))
	if len(errs) > 0 {
		for _, e := range errs {
			log.Errorf("Sync error: %s", e.Error())
		}
	}

	// Fin
	dumpContextToFile(c, c2)
	log.Info("ALL DONE -- Sync complete!")
}
