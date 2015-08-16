package conui

import (
	"core"
	"os"
	"time"

	"github.com/gizak/termui"
)

var (
	Redraw  = make(chan bool)
	Event   = termui.EventCh()
	Widgets = make(uiWidgetsMap)
)

// Initilizes the console GUI. termui.Close() must be called before exiting otherwise the terminal will not return to
// original state.
func Init() {
	os.Setenv("TERM", "xterm")
	err := termui.Init()
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			select {
			case e := <-Event:
				if e.Type == termui.EventKey && e.Ch == 'q' {
					termui.Close()
					os.Exit(0)
				}
				if e.Type == termui.EventResize {
					termui.Body.Width = termui.TermWidth()
					termui.Body.Align()
					go func() { Redraw <- true }()
				}
			case <-Redraw:
				termui.Render(termui.Body)
			}
		}
	}()
}

type uiWidgetsMap map[string]interface{}

func (w *uiWidgetsMap) widgetForDevice(d *core.Device) *DevicePanel {
	return nil
}

func BuildConsole(c *core.Context) {
	// Create the UI widgets
	Widgets["main"] = NewProgressGauge(2756489454684)
	Widgets["Test Disk 1"] = NewDevicePanel("Test Disk 1", 56548464684)

	// A 12 ccolumn grid is used to take up the entire terminal window
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0,
				Widgets["main"].(*ProgressGauge),
				Widgets["Test Disk 1"].(*DevicePanel),
			),
		),
	)
	termui.Body.Align()
}

func Update() {
	for {
		g1 := Widgets["main"].(*ProgressGauge)
		g1.SizeWritn += 141548485689
		g2 := Widgets["Test Disk 1"].(*DevicePanel)
		g2.SizeWritn += 1545464454
		Redraw <- true
		if g1.SizeWritn+1 > g1.SizeTotal {
			g1.SizeWritn = 1
		}
		if g2.SizeWritn+1 > g2.SizeTotal {
			g2.SizeWritn = 1
		}
		time.Sleep(time.Second / 10)
	}
}
