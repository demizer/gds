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
		// Not wrapped in fatal here because NewCatalog returns custom error types
		panic(err)
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
	visible := c.OutputStreamNum
	for x, y := range c.Devices {
		conui.Body.DevicePanels = append(conui.Body.DevicePanels, conui.NewDevicePanel(y.Name, y.SizeTotal))
		if visible > 0 {
			log.Debugln("Making device", x, "visible")
			conui.Body.DevicePanels[x].SetVisible(true)
			if x == 0 {
				conui.Body.DevicePanels[x].SetSelected(true)
			}
			visible--
		}
	}
	conui.Body.ProgressPanel = conui.NewProgressGauge(c.Catalog.TotalSize())
	conui.Body.ProgressPanel.SetVisible(true)
	conui.Layout()
}

func eventListener(c *core.Context) {
	defer cleanupAtExit()
	for {
		select {
		case e := <-conui.Events:
			if e.Type == conui.EventKey && e.Ch == 'j' {
				conui.Body.SelectNext()
			}
			if e.Type == conui.EventKey && e.Ch == 'k' {
				conui.Body.SelectPrevious()
			}
			if e.Type == conui.EventKey && e.Ch == 'd' {
				p := conui.Body.Selected()
				p.SetVisible(false)
			}
			if e.Type == conui.EventKey && e.Ch == 's' {
				p := conui.Body.Selected()
				p.SetVisible(true)
			}
			if e.Type == conui.EventKey && e.Key == conui.KeyEnter {
				p := conui.Body.Selected().Prompt()
				if p != nil {
					p.Action()
				}
			}
			if e.Type == conui.EventKey && e.Ch == 'q' {
				conui.Close()
				os.Exit(0)
			}
			if e.Type == conui.EventResize {
				// conui.Body.Width = conui.TermWidth()
				conui.Layout()
				go func() { conui.Redraw <- true }()
			}
		case <-conui.Redraw:
			conui.Render()
		}
	}
}

// deviceMountHandler checks to see if the device is mounted and writable. Meant to be run as a goroutine.
func deviceMountHandler(c *core.Context, deviceIndex int) {
	// Listen on the channel for a mount request
	ns := time.Now()
	log.Debugf("Waiting for receive on SyncDeviceMount[%d]", deviceIndex)
	<-c.SyncDeviceMount[deviceIndex]
	log.Debugf("Receive from SyncDeviceMount[%d] after wait of %s", deviceIndex, time.Since(ns))

	// Get the panel widget so we can write messages for the user to it
	wg := conui.Body.Select(deviceIndex)
	d, err := c.Devices.DeviceByName(wg.Border.Label)
	if err != nil {
		log.Error(err)
	}

	deviceIsReady := false
	checkDevice := func(p *conui.PromptAction) error {
		// The actual checking
		err := ensureDeviceIsReady(*d)
		if err != nil {
			log.Errorf("checkDevice error: %s", err)
			switch err.(type) {

			case deviceTestPermissionDeniedError:
				p.Message = "Device is mounted but not writable... " +
					"Please fix write permissions then press Enter to continue."
			case deviceNotFoundByUUIDError:
				p.Message = "Please mount device and press Enter to continue..."
			}
			return err
		}
		deviceIsReady = true
		return err
	}

	// The prompt that will be displayed in the device panel
	var pAction func()
	prompt := &conui.PromptAction{}
	// Allow the user to press enter on the device panel to force a device check
	pAction = func() {
		// With the device selected in the panel, the user has pressed the enter key.
		log.Printf("Action for panel %q!", wg.Border.Label)
		checkDevice(prompt)
	}
	prompt.Action = pAction

	// Finally, set the prompt for the device in the panel
	wg.SetPrompt(prompt)

	// Check device automatically periodically
	for {
		if deviceIsReady {
			break
		}
		// Rate limit
		err = checkDevice(prompt)
		if err != nil {
			time.Sleep(time.Second * 15)
			continue
		}
		break
	}
	// The prompt is not needed anymore
	wg.SetPrompt(nil)
	c.SyncDeviceMount[deviceIndex] <- true
}

func update(c *core.Context) {
	for x := 0; x < len(c.Devices); x++ {
		go deviceMountHandler(c, x)
		c.SyncDeviceMount[x] = make(chan bool)
		c.SyncProgress[x] = make(chan core.SyncProgress, 100)
		c.SyncFileProgress[x] = make(chan core.SyncFileProgress, 100)
		go func(index int) {
			lTime := time.Now()
			for {
				select {
				case p := <-c.SyncProgress[index]:
					prg := conui.Body.ProgressPanel
					prg.SizeWritn = p.ProgressSizeWritn
					dw := conui.Body.DevicePanelByIndex(index)
					dw.SizeWritn = p.DeviceSizeWritn
				case fp := <-c.SyncFileProgress[index]:
					if time.Since(lTime) > (time.Second / 5) {
						log.WithFields(logrus.Fields{
							"fp.FileName":     fp.FileName,
							"fp.SizeTotal":    fp.SizeTotal,
							"fp.FileWritn":    fp.SizeWritn,
							"fp.SizeWritnInc": fp.SizeWritnInc,
							"bps":             fp.BytesPerSecond,
						}).Debugln("Sync file progress")
						dw := conui.Body.DevicePanelByIndex(index)
						dw.SizeWritn += fp.SizeWritnInc
						lTime = time.Now()
					}
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

	// Give the user time to review the sync in the UI
	for {
		time.Sleep(time.Second)
	}
}
