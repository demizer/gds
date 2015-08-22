package conui

import (
	"io/ioutil"
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

type ConuiGridBufferer interface {
	termui.GridBufferer
	IsSelected() bool
	SetSelected(bool)
	IsVisible() bool
	SetVisible(bool)
	Prompt() *PromptAction
	SetPrompt(*PromptAction)
}

type uiWidgetsMap map[int]ConuiGridBufferer

func (w *uiWidgetsMap) selected() int {
	var x int
	var y ConuiGridBufferer
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
		(*w)[x].(*DevicePanel).SetSelected(false)
		(*w)[x].(*DevicePanel).Border.FgColor = termui.ColorWhite
	}
}

func (w *uiWidgetsMap) Select(index int) *DevicePanel {
	w.deselectAll()
	wg := (*w)[index].(*DevicePanel)
	Selected = index
	wg.SetSelected(true)
	wg.SetVisible(true)
	return wg
}

func (w *uiWidgetsMap) SelectPrevious() *DevicePanel {
	Selected = w.selected()
	w.deselectAll()
	var wg *DevicePanel
	var ok bool
	for {
		Selected--
		if Selected < 0 {
			Selected = len(*w) - 1
		}
		if wg, ok = (*w)[Selected].(*DevicePanel); ok {
			visible := wg.IsVisible()
			Log.Debugf("SELECTED previous Widget[%d].Visible = %t", Selected, visible)
			if visible {
				break
			}
		}
	}
	wg.SetSelected(true)
	return wg
}

func (w *uiWidgetsMap) SelectNext() *DevicePanel {
	Selected = w.selected()
	w.deselectAll()
	var wg *DevicePanel
	var ok bool
	for {
		Selected++
		if Selected > len(*w)-1 {
			Selected = 0
		}
		if wg, ok = (*w)[Selected].(*DevicePanel); ok {
			visible := wg.IsVisible()
			Log.Debugf("SELECTED next Widget[%d].Visible = %t", Selected, visible)
			if visible {
				break
			}
		}
	}
	wg.SetSelected(true)
	return wg
}

func (w *uiWidgetsMap) Selected() *DevicePanel {
	i := w.selected()
	return (*w)[i].(*DevicePanel)
}

func (w *uiWidgetsMap) DevicePanelByIndex(index int) *DevicePanel {
	return (*w)[index].(*DevicePanel)
}

func (w *uiWidgetsMap) PromptByIndex(index int, prompt PromptAction) *DevicePanel {
	wg := (*w)[index].(*DevicePanel)
	if wg != nil {
		Widgets.Select(index)
		wg.SetVisible(true)
	}
	return wg
}

// ProgressGauge returns the overall progress gauge from the widget list.
func (u *uiWidgetsMap) ProgressGauge() *ProgressGauge {
	return (*u)[len(*u)-1].(*ProgressGauge)
}

func Close() {
	if termbox.IsInit {
		termui.Close()
	}
}
