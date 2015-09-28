package conui

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"text/tabwriter"

	"github.com/demizer/go-humanize"
	"github.com/mattn/go-runewidth"
)

// DeviceFile is used by DevicePanel to track progress.
type DeviceFile struct {
	Name      string
	Path      string
	SizeWritn uint64
	SizeTotal uint64
	Bps       uint64 // Write speed in bytes per second
}

// DeviceFileHist is a helper type for tracking progress.
type DeviceFileHist []DeviceFile

// Append is used to append a new DeviceFile to the DeviceFileHist.
func (f *DeviceFileHist) Append(fl DeviceFile) {
	*f = append(*f, fl)
}

// UpdateLast should be used to update the last value in the DeviceFileHist.
func (f *DeviceFileHist) UpdateLast(bps uint64, sizeWritten uint64) {

}

// The size of the box drawn around the widget
var borderSize = 2

// DevicePanel widget for showing file sync progress to a device.
type DevicePanel struct {
	Border              labeledBorder  // Widget border dimensions
	Label               string         // Label of the widget
	SizeWritn           uint64         // Size of bytes written
	SizeTotal           uint64         // Total data size of the output
	DeviceFileHist      DeviceFileHist // Log of files seen
	FileHistoryViewable int            // Number of files to show in the history log

	// private
	visible  bool
	percent  int // The calculated percentage
	selected bool

	// Dimensions
	x                 int
	y                 int
	innerX            int
	innerY            int
	height            int
	width             int
	innerWidth        int
	innerHeight       int
	progressBarHeight int // The height of the progress bar

	// Prompt user to do things
	prompt *PromptAction
}

// NewGauge return a new DevicePanel with current theme.
func NewDevicePanel(label string, fileSize uint64) *DevicePanel {
	g := &DevicePanel{
		Label:               label,
		SizeWritn:           1,
		SizeTotal:           fileSize,
		FileHistoryViewable: 5,
		progressBarHeight:   2,
	}
	g.height = borderSize + g.progressBarHeight + g.FileHistoryViewable + 1
	g.Border.Height = g.height
	g.Border.Label = label
	return g
}

func (g *DevicePanel) IsSelected() bool {
	return g.selected
}

func (g *DevicePanel) SetSelected(b bool) {
	g.selected = b
}

func (g *DevicePanel) SetPrompt(p *PromptAction) {
	g.prompt = p
}

func (g *DevicePanel) IsVisible() bool {
	return g.visible
}

func (g *DevicePanel) SetVisible(b bool) {
	g.visible = b
}

func (g *DevicePanel) Prompt() *PromptAction {
	return g.prompt
}

// Buffer implements Bufferer interface.
func (g *DevicePanel) Buffer() []Point {
	if !g.visible {
		return nil
	}
	// update the border dimensions
	g.Border.X = g.x
	g.Border.Y = g.y
	g.Border.Width = g.width

	g.innerX = g.x + borderSize/2
	g.innerY = g.y + borderSize/2

	if g.selected {
		g.Border.FgColor = ColorGreen
	}

	// reset inner dims for new height
	g.innerWidth = g.width - borderSize
	g.innerHeight = g.height - borderSize

	// render the border
	ps := g.Border.Buffer()

	// Render the progress bar
	g.percent = int(math.Ceil(float64(g.SizeWritn) / float64(g.SizeTotal) * float64(100)))
	if g.SizeWritn < 100 {
		g.percent = 0
	}
	w := g.percent * g.innerWidth / 100
	for i := 0; i < g.progressBarHeight; i++ {
		for j := 0; j < w; j++ {
			p := Point{}
			p.X = g.innerX + j
			p.Y = g.innerY + i
			p.Ch = ' '
			p.Bg = ColorCyan
			if p.Bg == ColorDefault {
				p.Bg |= AttrReverse
			}
			ps = append(ps, p)
		}
	}

	// Render the percentage
	s := fmt.Sprintf("%s/%s (%s%%)", humanize.IBytes(g.SizeWritn), humanize.IBytes(g.SizeTotal), strconv.Itoa(g.percent))
	pry := g.y + (g.height-g.FileHistoryViewable)/2
	rs := []rune(s)
	pos := (g.width - runewidth.StringWidth(s)) / 2

	for i, v := range rs {
		p := Point{}
		p.X = pos + i
		p.Y = pry
		p.Ch = v
		p.Fg = ColorWhite
		if w+g.x+1 > pos+i {
			p.Bg = ColorCyan
			if p.Bg == ColorDefault {
				p.Bg |= AttrReverse
			}

		} else {
			p.Bg = ColorBlack
		}
		ps = append(ps, p)
	}

	g.innerHeight -= g.progressBarHeight
	g.innerY += g.progressBarHeight

	// Build tab formatted file history list
	var buf bytes.Buffer
	tw := new(tabwriter.Writer)
	tw.Init(&buf, 8, 0, 1, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tw)
	for i := g.FileHistoryViewable - 1; i >= 0; i-- {
		if len(g.DeviceFileHist) == 0 {
			break
		}
		f := g.DeviceFileHist[(len(g.DeviceFileHist)-1)-i]
		fmt.Fprintf(tw, "%s  \t%s/%s\t   %s\n", humanize.IBytes(f.Bps), humanize.IBytes(f.SizeWritn),
			humanize.IBytes(f.SizeTotal), f.Path)
	}
	tw.Flush()

	// Render the formatted file list
	i, k, j := 0, 0, 0
	rs = []rune(buf.String())
	for k < len(rs) {
		if rs[k] == '\n' {
			i++
			j = 0
			k++
			continue
		}
		pi := Point{}
		pi.X = g.innerX + j
		pi.Y = g.innerY + i
		pi.Ch = rs[k]
		pi.Bg = ColorBlack
		pi.Fg = ColorWhite
		if i == g.FileHistoryViewable {
			pi.Fg = ColorGreen
		}
		ps = append(ps, pi)
		k++
		j++
	}

	// Render the prompt if set
	if g.prompt != nil && len(g.prompt.Message) > 0 {
		rs := []rune(g.prompt.Message)
		for x := 0; x < len(g.prompt.Message); x++ {
			pt := Point{}
			pt.X = g.x + x + 2
			pt.Y = g.y + g.Border.Height - 2
			pt.Ch = rs[x]
			pt.Bg = ColorBlack
			pt.Fg = ColorRed
			ps = append(ps, pt)
		}
	}

	return g.chopOverflow(ps)
}

// GetHeight implements GridBufferer. It returns current height of the block.
func (d DevicePanel) GetHeight() int {
	return d.height
}

// SetX implements GridBufferer interface, which sets block's x position.
func (d *DevicePanel) SetX(x int) {
	d.x = x
}

// SetY implements GridBufferer interface, it sets y position for block.
func (d *DevicePanel) SetY(y int) {
	d.y = y
}

// SetWidth implements GridBuffer interface, it sets block's width.
func (d *DevicePanel) SetWidth(w int) {
	d.width = w
}

// Removes characters that are out-of-bounds of the widget.
func (d *DevicePanel) chopOverflow(ps []Point) []Point {
	nps := make([]Point, 0, len(ps))
	x := d.x
	y := d.y
	w := d.width
	h := d.height
	for _, v := range ps {
		if v.X >= x &&
			v.X < x+w &&
			v.Y >= y &&
			v.Y < y+h {
			nps = append(nps, v)
		}
	}
	return nps
}
