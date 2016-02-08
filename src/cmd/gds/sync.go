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

// When set to true all the go routines exit
var exit = make(chan bool)

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
	log.WithFields(logrus.Fields{"path": cPath}).Info("Using configuration file")

	c2, err := core.ContextFromPath(cPath)
	if err != nil {
		panic(fatal{fmt.Sprintf("Error loading config: %s", err.Error())})
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

// InitPanelUI creates the UI widgets First is the main progress guage for the overall progress Widgets are then created for
// each of the devices, but are hidden initially.
func InitPanelUI(c *core.Context) {
	visible := c.OutputStreamNum
	for x, y := range c.Devices {
		conui.Body.DevicePanels = append(conui.Body.DevicePanels, conui.NewDevicePanel(y.Name, y.SizeTotal))
		if visible > 0 && c.DevicesUsed > 0 {
			log.Debugln("Making device", x, "visible")
			conui.Body.DevicePanels[x].SetVisible(true)
			if x == 0 {
				conui.Body.DevicePanels[x].SetSelected(true)
			}
			visible--
		}
	}
	conui.Body.ProgressPanel = conui.NewProgressGauge(c.FileIndex.TotalSize())
	conui.Body.ProgressPanel.SetVisible(true)
	conui.Layout()
}

func eventHandler(c *core.Context) {
	defer cleanupAtExit()
	go func() {
		for {
			if !termbox.IsInit {
				break
			}
			conui.Redraw <- true
			// Rate limit redrawing
			time.Sleep(time.Second / 3)
		}
	}()
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
				log.Warnln("Sending signal to shutdown!")
				log.Errorln(conui.Body.HashingProgressGauge.SizeWritn, conui.Body.HashingProgressGauge.SizeTotal)
				select {
				case <-c.Done:
				default:
					log.Debugln("Closing done channel")
					close(c.Done)
				}
				conui.Close()
				close(exit)
				break
			}
			if e.Type == conui.EventResize {
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

	d := c.Devices[deviceIndex]
	wg := conui.Body.DevicePanelByIndex(deviceIndex)
	wg.SetVisible(true)

	// Used to correctly time the display of the message in the prompt for the device panel.
	pmc := make(chan string)

	checkDevice := func(p *conui.PromptAction, keyEvent bool, mesgChan chan string) (err error) {
		// The actual checking
		err = ensureDeviceIsReady(d)
		if err != nil {
			log.Errorf("checkDevice error: %s", err)
			switch err.(type) {
			case deviceTestPermissionDeniedError:
				p.Message = "Device is mounted but not writable... " +
					"Please fix write permissions then press Enter to continue."
			case deviceNotFoundByUUIDError:
				if keyEvent {
					pmc <- fmt.Sprintf("Device not found! (UUID=%s)", d.UUID)
				} else {
					pmc <- fmt.Sprintf("Please mount device to %q and press Enter to continue...",
						d.MountPoint)
				}
			}
			return
		}
		if deviceIndex == 0 {
			wg.SetSelected(true)
		}
		return
	}

	// The prompt that will be displayed in the device panel. Allow the user to press enter on the device panel to force
	// a device check
	prompt := &conui.PromptAction{}
	prompt.Action = func() {
		// With the device selected in the panel, the user has pressed the enter key.
		log.Printf("Action for panel %q!", wg.Border.Label)
		checkDevice(prompt, true, pmc)
	}
	wg.SetPrompt(prompt)

	// Pre-check
	err := checkDevice(prompt, false, pmc)
	if err != nil {
		// Check device automatically periodically until the device is mounted
	loop:
		for {
			// This will make sure the message is visible for a constant amount of time
			select {
			case pmsg := <-pmc:
				prompt.Message = pmsg
				conui.Redraw <- true
			case <-time.After(time.Second * 5):
				err := checkDevice(prompt, false, pmc)
				if err == nil {
					break loop
				}
			}
		}
	}

	// The prompt is not needed anymore
	wg.SetPrompt(nil)
	c.SyncDeviceMount[deviceIndex] <- true
}

func progressUpdater(c *core.Context) {
	// Main progress panel updater
	go func() {
		for {
			select {
			case p := <-c.SyncProgress.Report:
				prg := conui.Body.ProgressPanel
				prg.SizeWritn = p.SizeWritn
				prg.BytesPerSecond = p.BytesPerSecond
			case <-c.Done:
				return
			}
		}
	}()
	// Device panel updaters
	for x := 0; x < c.DevicesUsed; x++ {
		go deviceMountHandler(c, x)
		c.SyncDeviceMount[x] = make(chan bool)
		go func(index int) {
			dw := conui.Body.DevicePanelByIndex(index)
		outer:
			for {
				select {
				// The Report channel will be closed by the syncLaunch function once copying to the device
				// is complete.
				case fp, ok := <-c.SyncProgress.Device[index].Report:
					if !ok {
						break outer
					}
					dw.SizeWritn += fp.DeviceSizeWritn
					dw.BytesPerSecond = fp.DeviceBytesPerSecond
					log.WithFields(logrus.Fields{
						"fp.FileName":           fp.FileName,
						"fp.FilePath":           fp.FilePath,
						"fp.FileSize":           fp.FileSize,
						"fp.FileSizeWritn":      fp.FileSizeWritn,
						"fp.FileTotalSizeWritn": fp.FileTotalSizeWritn,
						"deviceIndex":           index,
					}).Debugln("Sync file progress")
				case <-c.Done:
					break
				}
			}
			dw.BytesPerSecondVisible = false
			log.Debugln("DONE REPORTING index:", index)
		}(x)
	}
}

func calcFileIndexHashes(c *core.Context) {
	h := core.NewSourceFileHashComputer(c.FileIndex, c.Errors)
	conui.Body.HashingProgressGauge = conui.NewHashingProgressGauge(c.FileIndex.TotalSizeFiles())
	conui.Body.HashingProgressGauge.SetVisible(true)
	go func() {
		conui.Body.HashingDialog = conui.NewHashingDialog(8, 2)
		bars := make(map[string]*conui.HashingProgressBar)
		bps := core.NewBytesPerSecond()
		for {
			select {
			case hf, ok := <-h.Reports:
				if !ok {
					return
				}
				bps.AddPoint(hf.SizeWritnLast)
				conui.Body.HashingProgressGauge.SizeWritn += hf.SizeWritnLast
				conui.Body.HashingProgressGauge.BytesPerSecond = bps.Calc()
				if conui.Body.HashingProgressGauge.SizeWritn == conui.Body.HashingProgressGauge.SizeTotal {
					conui.Body.HashingProgressGauge.BytesPerSecond = bps.CalcFull()
				}
				if hf.SizeWritn == hf.SizeTotal {
					log.WithFields(logrus.Fields{"filePath": hf.FilePath,
						"bytesWritnLast": hf.SizeWritnLast, "size": hf.SizeTotal,
					}).Debugln("calcFileIndexHashes: RECEIVED: FILE WRITE COMPLETE")
				} else {
					log.WithFields(logrus.Fields{"filePath": hf.FilePath,
						"bytesWritnLast": hf.SizeWritnLast, "size": hf.SizeTotal,
					}).Debugln("calcFileIndexHashes: RECEIVED")
				}
				if val, ok := bars[hf.FilePath]; ok {
					val.SizeWritn = hf.SizeWritn
					val.BytesPerSecond = hf.BytesPerSecond.Calc()
					if hf.SizeWritn == hf.SizeTotal {
						val.BytesPerSecond = hf.BytesPerSecond.CalcFull()
					}
				} else {
					bars[hf.FilePath] = conui.Body.HashingDialog.AddBar(hf.FileName, hf.SizeWritn, hf.SizeTotal)
					conui.Body.HashingDialog.SetVisible(true)
					conui.Layout()
				}
				conui.Body.HashingDialog.SortBars()
			case err := <-c.Errors:
				log.Error(err)
			case <-c.Done:
				return
			}
		}
	}()
	h.ComputeAll(c.Done)
	conui.Body.HashingDialog.SetVisible(false)
	conui.Body.HashingDialog.Bars = nil
}

func syncStart(c *cli.Context) {
	defer cleanupAtExit()

	log.WithFields(logrus.Fields{
		"version": 0.2,
		"date":    time.Now().Format(time.RFC3339),
	}).Infoln("Ghetto Device Storage")

	c2 := loadInitialState(c)

	conui.Init()
	go eventHandler(c2)

	calcFileIndexHashes(c2)

	InitPanelUI(c2)
	progressUpdater(c2)

	// log.Debugln(spd.Sdump(c2.FileIndex))
	// conui.Close()
	// os.Exit(0)

	// Sync the things
	go func() {
		core.Sync(c2, c.GlobalBool("no-dev-context"))
		log.Info("ALL DONE -- Sync complete!")
		// c2.Exit = true
	}()

	// Give the user time to review the sync in the UI
outer:
	for {
		select {
		case err := <-c2.Errors:
			log.Errorf("Sync error: %s", err)
		case <-exit:
			break outer
		}
	}

	// Fin
	dumpContextToFile(c, c2)
}
