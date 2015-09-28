// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package conui

// import tm "github.com/nsf/termbox-go"
// import rw "github.com/mattn/go-runewidth"

/* ---------------Port from termbox-go --------------------- */

// Attribute is printable cell's color and style.
type Attribute uint16

const (
	ColorDefault Attribute = iota
	ColorBlack
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

const NumberofColors = 8 //Have a constant that defines number of colors
const (
	AttrBold Attribute = 1 << (iota + 9)
	AttrUnderline
	AttrReverse
)
