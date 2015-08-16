package main

import (
	"conui"
	"core"
	"os"

	"github.com/gizak/termui"
)

var (
	uiRedraw = make(chan bool)
	uiEvent  = termui.EventCh()
)

// Initilizes the console GUI. termui.Close() must be called before exiting otherwise the terminal will not return to
// original state.
func init() {
	os.Setenv("TERM", "xterm")
	err := termui.Init()
	if err != nil {
		panic(err)
	}
	for {
		select {
		case e := <-uiEvent:
			if e.Type == termui.EventKey && e.Ch == 'q' {
				return
			}
			if e.Type == termui.EventResize {
				termui.Body.Width = termui.TermWidth()
				termui.Body.Align()
				go func() { uiRedraw <- true }()
			}
		case <-uiRedraw:
			termui.Render(termui.Body)
		}
	}
}

type uiWidgets map[string]interface{}

func (w *widgets) UIWidgetForDevice(d *core.Device) *conui.DevicePanel {

}

func uiBuildConsole(c *core.Context) {
	g1 := conui.NewProgressGauge(2756489454684)
	g2 := conui.NewDevicePanel("Test Disk 1", 56548464684)
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, g1, g2),
		),
	)
	termui.Body.Align()
	// termui.Render(termui.Body)
}

func uiUpdate() {
	for {
		g1.SizeWritn += 14154848568
		g2.SizeWritn += 1545464454
		uiRedraw <- true
		if g1.SizeWritn+1 > g1.SizeTotal {
			g1.SizeWritn = 1
		}
		if g2.SizeWritn+1 > g2.SizeTotal {
			g2.SizeWritn = 1
		}
		time.Sleep(time.Second / 10)
	}
}
