// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Copyright 2015 Jesus Alvarez <jeezusjr@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package conui

type Grid struct {
	Rows        []Widget
	Width       int
	X           int
	Y           int
	BgColor     Attribute
	SelectedRow int
}

func (g *Grid) selected() int {
	var x int
	var y Widget
	for x, y = range g.Rows {
		if y.IsSelected() {
			break
		}
	}
	return x
}

func (g *Grid) deselectAll() {
	for x, _ := range g.Rows {
		if _, ok := g.Rows[x].(*DevicePanel); !ok {
			continue
		}
		g.Rows[x].SetSelected(false)
		g.Rows[x].(*DevicePanel).Border.FgColor = ColorWhite
	}
}

func (g *Grid) NumVisible() int {
	num := 0
	for x := 1; x < len(g.Rows)-1; x++ {
		if g.Rows[x].(*DevicePanel).IsVisible() {
			num += 1
		}
	}
	return num
}

func (g *Grid) Select(index int) *DevicePanel {
	g.deselectAll()
	wg := g.Rows[index].(*DevicePanel)
	g.SelectedRow = index
	wg.SetSelected(true)
	wg.SetVisible(true)
	return wg
}

func (g *Grid) SelectPrevious() *DevicePanel {
	g.SelectedRow = g.selected()
	g.deselectAll()
	var wg *DevicePanel
	var ok bool
	for {
		g.SelectedRow--
		if g.SelectedRow < 0 {
			g.SelectedRow = len(g.Rows) - 1
		}
		if wg, ok = g.Rows[g.SelectedRow].(*DevicePanel); ok {
			visible := wg.IsVisible()
			Log.Debugf("SELECTED previous Widget[%d].Visible = %t", g.SelectedRow, visible)
			if visible {
				if (g.NumVisible() * g.Rows[1].Height()) >= TermHeight() {
					g.scrollVisible()
				}
				break
			}
		}
	}
	wg.SetSelected(true)
	return wg
}

func (g *Grid) scrollVisible() {
	deviceWidgetHeight := float64(g.Rows[1].Height())
	yPos := 0
	for x := 0; x < len(g.Rows); x++ {
		if !g.Rows[x].IsVisible() {
			continue
		}
		if g.SelectedRow == x {
			// Reset view
			yPos = g.Rows[0].Height()
		} else {
			yPos += int(deviceWidgetHeight)
		}
		if wg, ok := g.Rows[x].(*DevicePanel); ok {
			wg.SetY(yPos)
		}
	}
}

func (g *Grid) SelectNext() *DevicePanel {
	g.SelectedRow = g.selected()
	g.deselectAll()
	var wg *DevicePanel
	var ok bool
	for {
		g.SelectedRow++
		if g.SelectedRow > len(g.Rows)-1 {
			g.SelectedRow = 0
		}
		if wg, ok = g.Rows[g.SelectedRow].(*DevicePanel); ok {
			visible := wg.IsVisible()
			Log.Debugf("SELECTED next Widget[%d].Visible = %t", g.SelectedRow, visible)
			// Log.Debugf("%s", spd.Sdump(wg.Border))
			// Log.Debugln("TermHeight:", TermHeight(), "widget y:", wg.Border.Y, "widget height:", wg.Border.Height)
			if visible {
				Log.Debugf("termHeight: %d wg.Y: %d wg.Height: %d", TermHeight(), wg.Y(), wg.Height())
				// if (TermHeight()-wg.Y()) < wg.Height() || wg.Y() == 0 {
				if (g.NumVisible() * g.Rows[1].Height()) >= TermHeight() {
					g.scrollVisible()
				}
				break
			}
		}
	}
	wg.SetSelected(true)
	return wg
}

func (g *Grid) Selected() *DevicePanel {
	return g.Rows[g.SelectedRow].(*DevicePanel)
}

func (g *Grid) DevicePanelByIndex(index int) *DevicePanel {
	return g.Rows[index].(*DevicePanel)
}

func (g *Grid) PromptByIndex(index int, prompt PromptAction) *DevicePanel {
	wg := g.Rows[index].(*DevicePanel)
	if wg != nil {
		g.Select(index)
		wg.SetVisible(true)
	}
	return wg
}

// ProgressGauge returns the overall progress gauge from the widget list.
func (g *Grid) ProgressGauge() *ProgressGauge {
	return g.Rows[len(g.Rows)-1].(*ProgressGauge)
}

// NewGrid returns *Grid with given rows.
func NewGrid(rows ...Widget) *Grid {
	return &Grid{Rows: rows}
}

// AddRows appends given rows to Grid.
func (g *Grid) AddRows(rs ...Widget) {
	for _, y := range rs {
		g.Rows = append(g.Rows, y)
	}
}

// Buffer implments Bufferer interface.
func (g Grid) Buffer() []Point {
	ps := []Point{}
	// LIFO with the rows... row 0 has the highest priority, meaning the codes in row 0 cannot be overwritten.
	for x := len(g.Rows) - 1; x >= 0; x-- {
		ps = append(ps, g.Rows[x].Buffer()...)
	}
	return ps
}
