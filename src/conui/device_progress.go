package conui

import (
	"bytes"
	"fmt"
	"strconv"
	"text/tabwriter"

	"github.com/demizer/go-humanize"
	"github.com/gizak/termui"
	"github.com/mattn/go-runewidth"
)

// File is used by FileProgressGauge to track progress.
type File struct {
	Name      string
	Path      string
	SizeWritn uint64
	SizeTotal uint64
	Bps       uint64 // Write speed in bytes per second
}

// FileHistory is a helper type for tracking progress.
type FileHistory []File

// Append is used to append a new File to the FileHistory.
func (f *FileHistory) Append(fl File) {
	*f = append(*f, fl)
}

// UpdateLast should be used to update the last value in the FileHistory.
func (f *FileHistory) UpdateLast(bps uint64, sizeWritten uint64) {

}

// The size of the box drawn around the widget
var borderSize = 2

// FileProgressGauge widget for showing file sync progress to a device.
type FileProgressGauge struct {
	Border              labeledBorder // Widget border dimensions
	Label               string        // Label of the widget
	SizeWritn           uint64        // Size of bytes written
	SizeTotal           uint64        // Total data size of the output
	FileHistory         FileHistory   // Log of files seen
	FileHistoryViewable int           // Number of files to show in the history log
	percent             int           // The calculated percentage

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
}

// NewGauge return a new FileProgressGauge with current theme.
func NewFileProgressGauge(label string, fileSize uint64) *FileProgressGauge {
	g := &FileProgressGauge{
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

// Buffer implements Bufferer interface.
func (g *FileProgressGauge) Buffer() []termui.Point {
	// update the border dimensions
	g.Border.X = g.x
	g.Border.Y = g.y
	g.Border.Width = g.width

	g.innerX = g.x + borderSize/2
	g.innerY = g.y + borderSize/2

	// reset inner dims for new height
	g.innerWidth = g.width - borderSize
	g.innerHeight = g.height - borderSize

	// render the border
	ps := g.Border.Buffer()

	// Render the progress bar
	g.percent = int(float32(g.SizeWritn) / float32(g.SizeTotal) * float32(100))
	w := g.percent * g.width / 100
	for i := 0; i < g.progressBarHeight; i++ {
		for j := 0; j < w; j++ {
			p := termui.Point{}
			p.X = g.innerX + j
			p.Y = g.innerY + i
			p.Ch = ' '
			p.Bg = termui.ColorCyan
			if p.Bg == termui.ColorDefault {
				p.Bg |= termui.AttrReverse
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
		p := termui.Point{}
		p.X = pos + i
		p.Y = pry
		p.Ch = v
		p.Fg = termui.ColorWhite
		if w+g.x+1 > pos+i {
			p.Bg = termui.ColorCyan
			if p.Bg == termui.ColorDefault {
				p.Bg |= termui.AttrReverse
			}

		} else {
			p.Bg = termui.ColorBlack
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
		f := g.FileHistory[(len(g.FileHistory)-1)-i]
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
		pi := termui.Point{}
		pi.X = g.innerX + j
		pi.Y = g.innerY + i
		pi.Ch = rs[k]
		pi.Bg = termui.ColorBlack
		pi.Fg = termui.ColorWhite
		if i == g.FileHistoryViewable {
			pi.Fg = termui.ColorGreen
		}
		ps = append(ps, pi)
		k++
		j++
	}

	return g.chopOverflow(ps)
}

// GetHeight implements GridBufferer. It returns current height of the block.
func (d FileProgressGauge) GetHeight() int {
	return d.height
}

// SetX implements GridBufferer interface, which sets block's x position.
func (d *FileProgressGauge) SetX(x int) {
	d.x = x
}

// SetY implements GridBufferer interface, it sets y position for block.
func (d *FileProgressGauge) SetY(y int) {
	d.y = y
}

// SetWidth implements GridBuffer interface, it sets block's width.
func (d *FileProgressGauge) SetWidth(w int) {
	d.width = w
}

// Removes characters that are out-of-bounds of the widget.
func (d *FileProgressGauge) chopOverflow(ps []termui.Point) []termui.Point {
	nps := make([]termui.Point, 0, len(ps))
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
