package conui

import (
	"io/ioutil"
	"log"
	"logfmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/gizak/termui"
	"github.com/nsf/termbox-go"
)

var (
	Redraw   = make(chan bool)
	Event    = termui.EventCh()
	Widgets  = make(uiWidgetsMap)
	Selected = 1
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
func Init() {
	os.Setenv("TERM", "xterm")
	err := termui.Init()
	if err != nil {
		panic(err)
	}
}

type MyGridBufferer interface {
	termui.GridBufferer
	IsSelected() bool
}

type uiWidgetsMap map[int]MyGridBufferer

func (w *uiWidgetsMap) selected() int {
	var x int
	var y MyGridBufferer
	for x, y = range *w {
		if y.IsSelected() {
			break
		}
	}
	return x
}

func (w *uiWidgetsMap) deselectAll() {
	for x, _ := range *w {
		if _, ok := (*w)[x].(*DevicePanel); !ok {
			continue
		}
		(*w)[x].(*DevicePanel).Selected = false
		(*w)[x].(*DevicePanel).Border.FgColor = termui.ColorWhite
	}
}

func (w *uiWidgetsMap) Select(index int) *DevicePanel {
	w.deselectAll()
	wg := (*w)[index].(*DevicePanel)
	Selected = index
	wg.Selected = true
	return wg
}

func (w *uiWidgetsMap) SelectPrevious() *DevicePanel {
	Selected = w.selected()
	w.deselectAll()
	for {
		Selected--
		if Selected < 0 {
			Selected = len(*w) - 2
		}
		Log.Debugln("SELECTED previous:", Selected)
		if (*w)[Selected].(*DevicePanel).Visible {
			break
		}
	}
	(*w)[Selected].(*DevicePanel).Selected = true
	return (*w)[Selected].(*DevicePanel)
}

func (w *uiWidgetsMap) SelectNext() *DevicePanel {
	Selected = w.selected()
	w.deselectAll()
	for {
		Selected++
		if Selected == len(*w)-1 || Selected > len(*w)-1 {
			Selected = 0
		}
		Log.Debugln("SELECTED next:", Selected)
		if (*w)[Selected].(*DevicePanel).Visible {
			break
		}
	}
	(*w)[Selected].(*DevicePanel).Selected = true
	return (*w)[Selected].(*DevicePanel)
}

func (w *uiWidgetsMap) Selected() *DevicePanel {
	i := w.selected()
	return (*w)[i].(*DevicePanel)
}

func (w *uiWidgetsMap) MountPromptByIndex(index int) *DevicePanel {
	wg := (*w)[index].(*DevicePanel)
	if wg != nil {
		Widgets.Select(index)
		wg.Visible = true
		wg.Prompt = Prompt{
			Message: "Please mount device and press Enter to continue...",
			Action: func() {
				log.Printf("Action for %s!!", wg.Border.Label)
			},
		}
	}
	return wg
}

func (w *uiWidgetsMap) MountPromptByName(name string) (int, *DevicePanel) {
	var index int
	var wg *DevicePanel
	for x, y := range *w {
		switch dp := y.(type) {
		case *DevicePanel:
			if dp.Border.Label == name {
				index = x
				wg = (*w)[x].(*DevicePanel)
			}
		}
	}
	if wg != nil {
		Widgets.Select(index)
		wg.Visible = true
		wg.Prompt = Prompt{
			Message: "Please mount device and press Enter to continue...",
			Action: func() {
				log.Printf("Action for %s!!", wg.Border.Label)
			},
		}
	}
	return index, wg
}

func Close() {
	if termbox.IsInit {
		termui.Close()
	}
}
