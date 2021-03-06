package conui

import (
	"io/ioutil"
	"logfmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/davecgh/go-spew/spew"
	"github.com/nsf/termbox-go"
)

var spd = spew.ConfigState{Indent: "\t"} //, DisableMethods: true}

var (
	Body   *Grid
	Redraw = make(chan bool)
	Events = EventCh()
)

// Log is the default logging object. By default, all output is discarded. Set Log.Out to std.Stdout to enable output. The
// level of the log output can also be set in this manner. See the documentation of the logrus package for other options.
var Log = &logrus.Logger{
	Out:       ioutil.Discard,
	Formatter: new(logfmt.TextFormatter),
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.InfoLevel,
}

// Initilizes the console GUI. termui.Close() must be called before exiting otherwise the terminal will not return to
// original state.
//
// Init initializes termui library. This function should be called before any others.
// After initialization, the library must be finalized by 'Close' function.
func Init() error {
	os.Setenv("TERM", "xterm")
	err := termbox.Init()
	w, h := termbox.Size()
	Body = NewGrid(w, h)
	evtListen()
	return err
}

func Close() {
	if termbox.IsInit {
		termbox.Close()
	}
}

func Layout() {
	Log.WithFields(logrus.Fields{
		"NumVisible":               Body.NumVisible(),
		"TermHeight":               TermHeight(),
		"TermWidth":                TermWidth(),
		"Body.ProgressPanelHeight": Body.ProgressPanelHeight,
	}).Debugf("Layout")

	// Use all of the width of the terminal
	Body.Width = TermWidth()

	if Body.ProgressPanel.IsVisible() {
		Body.ProgressPanel.SetWidth(TermWidth())
	}
	if len(Body.DevicePanels) > 0 {
		for x := 0; x < len(Body.DevicePanels); x++ {
			Body.DevicePanels[x].SetWidth(TermWidth())
		}
		if (Body.NumVisible() * Body.DevicePanelHeight) > (TermHeight() - Body.ProgressPanelHeight) {
			Log.Debugln("Layout: Calling Body.scrollVisible()")
			Body.scrollVisible()
		} else {
			Log.Debugln("Layout: Default rendering")
			yPos := Body.ProgressPanelHeight
			Body.ProgressPanel.SetWidth(TermWidth())
			Body.ProgressPanel.SetY(0)
			for x := 0; x < len(Body.DevicePanels); x++ {
				row := Body.DevicePanels[x]
				row.SetWidth(TermWidth())
				yHeight := row.Height()
				row.SetY(yPos)
				yPos += yHeight
			}
		}
	}
}
