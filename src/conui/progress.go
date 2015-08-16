package conui

import (
	"fmt"
	"strconv"

	"github.com/demizer/go-humanize"
	"github.com/gizak/termui"
	"github.com/mattn/go-runewidth"
)

// ProgressGauge shows the total device sync progress
type ProgressGauge struct {
	termui.Block
	SizeWritn uint64 // Number of bytes written
	SizeTotal uint64 // The total size of the operation.
	FilePath  string
	percent   int
}

// NewProgressGauge returns an new ProgressGauge.
func NewProgressGauge(sizeTotal uint64) *ProgressGauge {
	g := &ProgressGauge{
		Block:     *termui.NewBlock(),
		SizeWritn: 1,
		SizeTotal: sizeTotal,
	}

	g.Width = 12
	g.Height = 5
	return g
}

// Buffer implements Bufferer interface.
func (g *ProgressGauge) Buffer() []termui.Point {
	ps := g.Block.Buffer()
	innerX, innerY, innerWidth, innerHeight := g.Block.InnerBounds()
	g.percent = int(float32(g.SizeWritn) / float32(g.SizeTotal) * float32(100))

	// plot bar
	w := g.percent * innerWidth / 100
	for i := 0; i < innerHeight; i++ {
		for j := 0; j < w; j++ {
			p := termui.Point{}
			p.X = innerX + j
			p.Y = innerY + i
			p.Ch = ' '
			p.Bg = termui.ColorCyan
			if p.Bg == termui.ColorDefault {
				p.Bg |= termui.AttrReverse
			}
			ps = append(ps, p)
		}
	}

	// plot percentage
	s := fmt.Sprintf("%s/%s (%s%%)", humanize.IBytes(g.SizeWritn), humanize.IBytes(g.SizeTotal), strconv.Itoa(g.percent))
	pry := innerY + innerHeight/2
	rs := []rune(s)
	pos := (innerWidth - runewidth.StringWidth(s)) / 2

	for i, v := range rs {
		p := termui.Point{}
		p.X = pos + i
		p.Y = pry
		p.Ch = v
		p.Fg = termui.ColorWhite
		if w+innerX > pos+i {
			p.Bg = termui.ColorCyan
			if p.Bg == termui.ColorDefault {
				p.Bg |= termui.AttrReverse
			}
		} else {
			p.Bg = g.Block.BgColor
		}

		ps = append(ps, p)
	}
	return ps
}
