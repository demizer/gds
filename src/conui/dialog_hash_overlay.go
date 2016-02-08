package conui

import (
	"fmt"
	"math"
	"strconv"

	"github.com/demizer/go-humanize"
	"github.com/nsf/termbox-go"
)

// HashingProgressGauge shows the total device sync progress
type HashingProgressGauge struct {
	Border         labeledBorder // Widget border dimensions
	FilePath       string
	SizeWritn      uint64 // Number of bytes written
	SizeTotal      uint64 // The total size of the operation.
	BytesPerSecond uint64 // The bytes per second

	visible bool
	percent int

	// Block
	x           int
	y           int
	width       int
	height      int
	innerX      int
	innerY      int
	innerWidth  int
	innerHeight int
}

// NewHashingProgressGauge returns an new HashingProgressGauge.
func NewHashingProgressGauge(sizeTotal uint64) *HashingProgressGauge {
	return &HashingProgressGauge{SizeTotal: sizeTotal}
}

func (g *HashingProgressGauge) IsVisible() bool { return g.visible }

func (g *HashingProgressGauge) SetVisible(b bool) { g.visible = b }

func (g *HashingProgressGauge) Width() int { return g.width }

func (g *HashingProgressGauge) SetWidth(w int) { g.width = w }

func (g *HashingProgressGauge) Height() int { return g.height }

func (g *HashingProgressGauge) SetHeight(h int) { g.height = h }

func (g *HashingProgressGauge) X() int { return g.x }

func (g *HashingProgressGauge) SetX(x int) { g.x = x }

func (g *HashingProgressGauge) Y() int { return g.y }

func (g *HashingProgressGauge) SetY(y int) { g.y = y }

// Buffer implements Bufferer interface.
func (g *HashingProgressGauge) Buffer() []Point {
	w, _ := termbox.Size()
	g.width = w - (w / 3)
	g.x = (w / 2) - (g.width / 2)
	g.height = 3
	g.height = borderSize + g.height
	g.Border.Height = g.height

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

	g.percent = int(math.Ceil(float64(g.SizeWritn) / float64(g.SizeTotal) * float64(100)))
	if g.SizeWritn < 100 {
		g.percent = 0
	}
	// Clear inside of panel
	for i := 0; i < g.innerHeight; i++ {
		for j := 0; j < g.innerWidth; j++ {
			ps = append(ps, Point{X: g.innerX + j, Y: g.innerY + i, Ch: ' '})
		}
	}
	// plot bar
	bw := g.percent * g.innerWidth / 100
	for i := 0; i < g.innerHeight; i++ {
		for j := 0; j < bw; j++ {
			p := Point{X: g.innerX + j, Y: g.innerY + i, Ch: ' ', Bg: ColorCyan}
			if p.Bg == ColorDefault {
				p.Bg |= AttrReverse
			}
			ps = append(ps, p)
		}
	}

	// plot percentage
	s := fmt.Sprintf("Computing SHA1 Hash ... %s/%s [%s/s] (%s%%)", humanize.IBytes(g.SizeWritn), humanize.IBytes(g.SizeTotal),
		humanize.IBytes(g.BytesPerSecond), strconv.Itoa(g.percent))
	pry := g.innerY + g.innerHeight/2
	rs := []rune(s)
	pos := (g.width / 2) //- (runewidth.StringWidth(s) / 2)

	for i, v := range rs {
		p := Point{}
		p.X = pos + i
		p.Y = pry
		p.Ch = v
		p.Fg = ColorWhite
		if bw+g.x+1 > pos+i {
			p.Bg = ColorCyan
			if p.Bg == ColorDefault {
				p.Bg |= AttrReverse
			}
		} else {
			p.Bg = ColorBlack
		}

		ps = append(ps, p)
	}
	return ps
}
