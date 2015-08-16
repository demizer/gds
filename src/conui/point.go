package conui

import (
	"github.com/gizak/termui"
)

// newPointWithAttrs creates a new termui point with the provided attributes.
func newPointWithAttrs(char rune, x, y int, fg, bg termui.Attribute) termui.Point {
	return termui.Point{
		Ch: char,
		X:  x,
		Y:  y,
		Bg: bg,
		Fg: fg,
	}
}
