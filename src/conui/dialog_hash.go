package conui

import (
	"fmt"
	"math"
	"time"

	"github.com/demizer/go-humanize"
	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

type HashingProgressBar struct {
	Label          string
	SizeWritn      uint64 // Size of bytes written
	SizeTotal      uint64 // Total data size of the output
	BytesPerSecond uint64 // Write speed in bytes per second
	height         int
	width          int
	x              int
	y              int
	percent        int // The calculated percentage
	timeFinished   time.Time
	sorted         bool
}

func (h *HashingProgressBar) String() string {
	return fmt.Sprintf("{Label: %s, SizeWritn: %d, SizeTotal: %d, BytesPerSecond: %d, sorted: %t}",
		h.Label, h.SizeWritn, h.SizeTotal, h.BytesPerSecond, h.sorted)
}

func (h *HashingProgressBar) Percent() int {
	h.percent = int(math.Ceil(float64(h.SizeWritn) / float64(h.SizeTotal) * float64(100)))
	return h.percent
}

func (h *HashingProgressBar) Text() string {
	return fmt.Sprintf("%s", h.Label)
}

func (h *HashingProgressBar) Stats() string {
	if h.SizeWritn == h.SizeTotal && h.timeFinished.IsZero() {
		Log.Debugf("Setting timeFinished for File: %q Size: %d", h.Label, h.SizeTotal)
		h.timeFinished = time.Now()
	}
	return fmt.Sprintf("%s/%s [%s/s] (%02.f%%)", humanize.IBytes(h.SizeWritn), humanize.IBytes(h.SizeTotal),
		humanize.IBytes(h.BytesPerSecond), float32(h.percent))
}

// BarWidth returns the width of the actual progress bar
func (h *HashingProgressBar) barWidth() int {
	return h.percent * h.width / 100
}

func (h *HashingProgressBar) BufferBar(ps *[]Point) {
	h.Percent()
	for i := 0; i < h.height; i++ {
		for j := 0; j < h.barWidth(); j++ {
			p := Point{
				X:  h.x + j,
				Y:  h.y,
				Ch: ' ',
				Bg: ColorBlue,
			}
			if p.Bg == ColorDefault {
				p.Bg |= AttrReverse
			}
			*ps = append(*ps, p)
		}
	}
}

func (h *HashingProgressBar) BufferLabel(ps *[]Point) {
	s := h.Text()
	pos := runewidth.StringWidth(s)
	for i, v := range []rune(s) {
		p := Point{
			X:  h.x + i + 1,
			Y:  h.y,
			Ch: v,
			Fg: ColorWhite,
		}
		if (h.barWidth() + h.x + pos - 1) > (h.x + pos + i) {
			p.Bg = ColorBlue
			if p.Bg == ColorDefault {
				p.Bg |= AttrReverse
			}

		} else {
			p.Bg = ColorBlack
		}
		*ps = append(*ps, p)
	}
}

func (h *HashingProgressBar) BufferStats(ps *[]Point) {
	s := h.Stats()
	pos := h.width - runewidth.StringWidth(s) - 1
	for i, v := range []rune(s) {
		p := Point{
			X:  h.x + pos + i,
			Y:  h.y,
			Ch: v,
			Fg: ColorWhite,
		}
		if (h.barWidth() + h.x) > (h.x + pos + i) {
			p.Bg = ColorBlue
			if p.Bg == ColorDefault {
				p.Bg |= AttrReverse
			}

		} else {
			p.Bg = ColorBlack
		}
		*ps = append(*ps, p)
	}
}

// HashingDialog widget for showing file sync progress to a device.
type HashingDialog struct {
	Border border // Widget border dimensions

	Bars       []*HashingProgressBar
	lastSorted int

	visible  bool
	selected bool

	columns int

	borderSize int
	barHeight  int // The height of the progress bars

	x      int
	y      int
	innerX int
	innerY int

	height      int
	width       int
	innerWidth  int
	innerHeight int
}

// NewGauge return a new HashingDialog with current theme. Use n to set the number of progress bars visible and c is the
// number of columns to display.
func NewHashingDialog(n int, c int) *HashingDialog {
	h := &HashingDialog{
		columns:    c,
		borderSize: 1,
		barHeight:  1,
	}
	return h
}

func (g *HashingDialog) setDimensions() {
	w, h := termbox.Size()
	g.barHeight = 1
	g.width = w - (w / 3)
	g.height = g.borderSize + (len(g.Bars) * g.barHeight)
	g.SetX((w / 2) - (g.width / 2))
	g.SetY(h - g.height)
	x := 0
	for _, b := range g.Bars {
		b.width = g.width - 4
		b.x = g.x + 2
		b.y = g.y + (x * g.barHeight) + 1
		x++
	}
	g.Border.X = g.x
	g.Border.Y = g.y
}

// Removes characters that are out-of-bounds of the widget.
func (d *HashingDialog) chopOverflow(ps []Point) []Point {
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

func (g *HashingDialog) IsVisible() bool {
	return g.visible
}

func (g *HashingDialog) SetVisible(b bool) {
	g.visible = b
}

func (g *HashingDialog) X() int { return g.x }

func (g *HashingDialog) SetX(x int) { g.x = x }

func (g *HashingDialog) Y() int { return g.y }

func (g *HashingDialog) SetY(y int) { g.y = y }

func (d HashingDialog) GetHeight() int { return d.height }

func (g *HashingDialog) AddBar(text string, sw uint64, st uint64) *HashingProgressBar {
	h := &HashingProgressBar{
		Label:     text,
		SizeWritn: sw,
		SizeTotal: st,
		height:    g.barHeight,
		x:         g.x,
		y:         g.y,
	}
	g.Bars = append(g.Bars, h)
	return h
}

// Buffer implements Bufferer interface.
func (g *HashingDialog) Buffer() []Point {
	if !g.visible {
		return nil
	}
	g.setDimensions()
	ps := g.Border.Buffer() // render the border

	// Render stuff
	for _, b := range g.Bars {
		b.BufferBar(&ps)
		b.BufferLabel(&ps)
		b.BufferStats(&ps)
	}

	return g.chopOverflow(ps)
}

// SortBars moves completed bars to the top of the current active bars keeping order as much as possible.
func (g *HashingDialog) SortBars() {
	// See commit Ð5b0ad75b60cfcfc25fb43b9ac583ab3948169 for debug logging, it has been removed because it is too
	// noisy/messy
	g.lastSorted = -1
	for x := 0; x < len(g.Bars); x++ {
		if g.Bars[x].sorted {
			g.lastSorted = x
			continue
		}
		f := g.Bars[x]
		// Move complete bars to after the previously found complete bar
		if f.SizeWritn == f.SizeTotal && !f.timeFinished.IsZero() && !f.sorted {
			if x == 0 || x > 0 && g.Bars[x-1].sorted {
				// The first bar (or current bar) is done, but not sorted
				g.lastSorted = x
				f.sorted = true
				continue

			}
			// Push down bars
			g.lastSorted++
			buf := g.Bars[g.lastSorted]
			g.Bars[g.lastSorted] = f
			for y := x; y >= g.lastSorted; y-- {
				if y == g.lastSorted+1 {
					g.Bars[y] = buf
					break
				}
				g.Bars[y] = g.Bars[y-1]
			}
			f.sorted = true
		}
	}
}
