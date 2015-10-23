package conui

import (
	"fmt"
	"math"
	"strconv"

	"github.com/demizer/go-humanize"
	"github.com/mattn/go-runewidth"
)

// ProgressGauge shows the total device sync progress
type ProgressGauge struct {
	Border    labeledBorder // Widget border dimensions
	SizeWritn uint64        // Number of bytes written
	SizeTotal uint64        // The total size of the operation.
	FilePath  string

	selected bool
	visible  bool
	percent  int
	prompt   *PromptAction

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

// NewProgressGauge returns an new ProgressGauge.
func NewProgressGauge(sizeTotal uint64) *ProgressGauge {
	g := &ProgressGauge{
		// Block:     *NewBlock(),
		SizeWritn: 1,
		SizeTotal: sizeTotal,
	}
	g.width = TermWidth()
	g.height = 3
	g.height = borderSize + g.height
	g.Border.Height = g.height
	return g
}

func (g *ProgressGauge) IsSelected() bool {
	return g.selected
}

func (g *ProgressGauge) SetSelected(b bool) {
	g.selected = b
}

func (g *ProgressGauge) IsVisible() bool {
	return g.visible
}

func (g *ProgressGauge) SetVisible(b bool) {
	g.visible = b
}

func (g *ProgressGauge) SetPrompt(p *PromptAction) {
	g.prompt = p
}

func (g *ProgressGauge) Prompt() *PromptAction {
	return g.prompt
}

func (g *ProgressGauge) Width() int { return g.width }

func (g *ProgressGauge) SetWidth(w int) { g.width = w }

func (g *ProgressGauge) Height() int { return g.height }

func (g *ProgressGauge) SetHeight(h int) { g.height = h }

func (g *ProgressGauge) X() int { return g.x }

func (g *ProgressGauge) SetX(x int) { g.x = x }

func (g *ProgressGauge) Y() int { return g.y }

func (g *ProgressGauge) SetY(y int) { g.y = y }

// Buffer implements Bufferer interface.
func (g *ProgressGauge) Buffer() []Point {
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

	g.percent = int(math.Ceil(float64(g.SizeWritn) / float64(g.SizeTotal) * float64(100)))
	if g.SizeWritn < 100 {
		g.percent = 0
	}
	// plot bar
	w := g.percent * g.innerWidth / 100
	for i := 0; i < g.innerHeight; i++ {
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

	// plot percentage
	s := fmt.Sprintf("%s/%s (%s%%)", humanize.IBytes(g.SizeWritn), humanize.IBytes(g.SizeTotal), strconv.Itoa(g.percent))
	pry := g.innerY + g.innerHeight/2
	rs := []rune(s)
	pos := (g.width - runewidth.StringWidth(s)) / 2

	for i, v := range rs {
		p := Point{}
		p.X = pos + i
		p.Y = pry
		p.Ch = v
		p.Fg = ColorWhite
		if w+g.x > pos+i {
			p.Bg = ColorCyan
			if p.Bg == ColorDefault {
				p.Bg |= AttrReverse
			}
			// } else {
			// p.Bg = g.BgColor
		}

		ps = append(ps, p)
	}
	return ps
}
