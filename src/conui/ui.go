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
	Body = NewGrid()
	Body.X = 0
	Body.Y = 0
	Body.SelectedRow = 1
	Body.ProgressPanelHeight = 5
	Body.DevicePanelHeight = 10
	Body.BgColor = theme.BodyBg

	err := termbox.Init()
	w, _ := termbox.Size()
	Body.Width = w
	evtListen()
	return err
}

func Close() {
	if termbox.IsInit {
		termbox.Close()
	}
}

func Layout() {
	yPos := 0
	Log.Debugf("Layout: NumVisible: %d TermHeight: %d Body.Rows[0].Height: %d", Body.NumVisible(), TermHeight(),
		Body.Rows[0].Height())
	if (Body.NumVisible() * Body.Rows[1].Height()) > (TermHeight() - 5) {
		Log.Debugln("Layout: Calling Body.scrollVisible()")
		Body.scrollVisible()
	} else {
		Log.Debugln("Layout: Default rendering")
		for x := 0; x < len(Body.Rows); x++ {
			row := Body.Rows[x]
			yHeight := row.Height()
			if x == len(Body.Rows)-1 {
				// Rows[0] is the overall progress row, so draw it last
				row.SetY(0)
			} else {
				row.SetY(yPos)
			}
			yPos += yHeight
		}
	}
}
