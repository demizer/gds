// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package conui

import "github.com/nsf/termbox-go"

// Bufferer should be implemented by all renderable components.
type Bufferer interface {
	Buffer() []Point
}

// TermWidth returns the current terminal's width.
func TermWidth() int {
	termbox.Sync()
	w, _ := termbox.Size()
	return w
}

// TermHeight returns the current terminal's height.
func TermHeight() int {
	termbox.Sync()
	_, h := termbox.Size()
	return h
}

func Render(rs ...Bufferer) {
	termbox.Clear(termbox.ColorDefault, termbox.Attribute(theme.BodyBg))
	for _, r := range rs {
		buf := r.Buffer()
		for _, v := range buf {
			termbox.SetCell(v.X, v.Y, v.Ch, termbox.Attribute(v.Fg), termbox.Attribute(v.Bg))
		}
	}
	termbox.Flush()
}
