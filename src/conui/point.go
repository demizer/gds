// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package conui

// Point stands for a single cell in terminal.
type Point struct {
	Ch rune
	Bg Attribute
	Fg Attribute
	X  int
	Y  int
}

func newPoint(c rune, x, y int) (p Point) {
	p.Ch = c
	p.X = x
	p.Y = y
	return
}

// newPointWithAttrs creates a new termui point with the provided attributes.
func newPointWithAttrs(char rune, x, y int, fg, bg Attribute) Point {
	return Point{
		Ch: char,
		X:  x,
		Y:  y,
		Bg: bg,
		Fg: fg,
	}
}
