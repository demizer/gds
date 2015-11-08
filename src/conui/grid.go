// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Copyright 2015 Jesus Alvarez <jeezusjr@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package conui

type Grid struct {
	Rows                []Widget
	Width               int
	X                   int
	Y                   int
	BgColor             Attribute
	SelectedDevicePanel int
	ProgressPanelHeight int
	DevicePanelHeight   int
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
	g.SelectedDevicePanel = index
	wg.SetSelected(true)
	wg.SetVisible(true)
	return wg
}

func (g *Grid) scrollVisible() {
	visibleArea := TermHeight() - 5

	Log.Debugf("scrollVisible: visibleArea: %d g.SelectedDevicePanel: %d numVisibleWidgets: %d",
		visibleArea, g.SelectedDevicePanel, g.NumVisible())

	if (TermHeight() - g.ProgressPanelHeight) < g.DevicePanelHeight {
		Log.Debugln("scrollVisible: Rendering only selected panel (not enough term height)")
		for x := 1; x < len(g.Rows)-1; x++ {
			if dp := g.DevicePanelByIndex(x); dp != nil {
				if x == g.SelectedDevicePanel {
					dp.SetY(g.ProgressPanelHeight)
					continue
				}
			}
		}
	} else if g.SelectedDevicePanel == 1 {
		Log.Debugln("scrollVisible: First visible panel selected")
		yPos := 5
		for x := 1; x < len(g.Rows)-1; x++ {
			if dp := g.DevicePanelByIndex(x); dp != nil {
				if !dp.IsVisible() {
					continue
				}
				dp.SetY(yPos)
				yPos += g.DevicePanelHeight
			}
		}
	} else if g.DevicePanelByIndexNextVisible(g.SelectedDevicePanel) == g.SelectedDevicePanel {
		Log.Debugln("scrollVisible: Last visible panel selected")
		yPos := TermHeight() - g.DevicePanelHeight
		for x := len(g.Rows) - 1; x > 0; x-- {
			if dp := g.DevicePanelByIndex(x); dp != nil {
				if !dp.IsVisible() {
					continue
				}
				dp.SetY(yPos)
				yPos -= g.DevicePanelHeight
			}
		}
	} else {
		Log.Debugln("scrollVisible: In-between panel selected")
		if dp := g.DevicePanelByIndex(g.SelectedDevicePanel); dp != nil {
			if (dp.Y()+g.DevicePanelHeight) < TermHeight() && dp.Y() > g.ProgressPanelHeight {
				Log.Debugln("scrollVisible: SelectedDevicePanel is fully visible, not doing anything")
				return
			}
		}
		var visiblePanels []int
		// The selected device will be set to the middle of the visible area
		var selRowInVis int
		for x := 1; x < len(g.Rows)-1; x++ {
			if dp := g.DevicePanelByIndex(x); dp != nil {
				if dp.IsVisible() {
					visiblePanels = append(visiblePanels, x)
					if x == g.SelectedDevicePanel {
						selRowInVis = len(visiblePanels) - 1
					}
				}
			}
		}
		Log.Debugf("scrollVisible: visiblePanels: %+v g.SelectedDevicePanel: %d", visiblePanels, g.SelectedDevicePanel)
		yPos := ((visibleArea / 2) - (g.DevicePanelHeight / 2)) + g.ProgressPanelHeight
		for x := 0; x < len(visiblePanels); x++ {
			if dp := g.DevicePanelByIndex(visiblePanels[x]); dp != nil {
				if x < selRowInVis {
					Log.Debugln("scrollVisible:", selRowInVis-x, "rows before from g.SelectedDevicePanel")
					ny := yPos - ((selRowInVis - x) * g.DevicePanelHeight)
					Log.Debugf("scrollVisible: visiblePanels[%d] = %d", x, visiblePanels[x])
					Log.Debugf("scrollVisible: g.Rows[%d].Y = %d", visiblePanels[x], ny)
					dp.SetY(ny)
					continue
				} else if x == selRowInVis {
					Log.Debugf("scrollVisible: g.Rows[%d].Y = %d", g.SelectedDevicePanel, yPos)
					dp.SetY(yPos)
				} else {
					Log.Debugln("scrollVisible:", x+1, "rows after g.SelectedDevicePanel")
					dp.SetY(yPos)
				}
				yPos += g.DevicePanelHeight
			}
		}
	}
}

func (g *Grid) SelectPrevious() *DevicePanel {
	g.SelectedDevicePanel = g.selected()
	g.deselectAll()
	var wg *DevicePanel
	var ok bool
	for {
		g.SelectedDevicePanel--
		if g.SelectedDevicePanel < 0 {
			g.SelectedDevicePanel = len(g.Rows) - 1
		}
		if wg, ok = g.Rows[g.SelectedDevicePanel].(*DevicePanel); ok {
			visible := wg.IsVisible()
			Log.Debugf("SELECTED previous Widget[%d].Visible = %t", g.SelectedDevicePanel, visible)
			if visible {
				Log.Debugf("SelectPrevious: termHeight: %d wg.Y: %d wg.Height: %d", TermHeight(), wg.Y(), wg.Height())
				if (g.NumVisible() * g.Rows[1].Height()) > (TermHeight() - 5) {
					g.scrollVisible()
				}
				break
			}
		}
	}
	wg.SetSelected(true)
	return wg
}

func (g *Grid) SelectNext() *DevicePanel {
	g.SelectedDevicePanel = g.selected()
	g.deselectAll()
	var wg *DevicePanel
	var ok bool
	for {
		g.SelectedDevicePanel++
		if g.SelectedDevicePanel > len(g.Rows)-1 {
			g.SelectedDevicePanel = 0
		}
		if wg, ok = g.Rows[g.SelectedDevicePanel].(*DevicePanel); ok {
			visible := wg.IsVisible()
			Log.Debugf("SelectNext: Widget[%d].Visible = %t", g.SelectedDevicePanel, visible)
			if visible {
				Log.Debugf("SelectNext: termHeight: %d wg.Y: %d wg.Height: %d", TermHeight(), wg.Y(), wg.Height())
				if (g.NumVisible() * g.Rows[1].Height()) > (TermHeight() - 5) {
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
	return g.Rows[g.SelectedDevicePanel].(*DevicePanel)
}

func (g *Grid) DevicePanelByIndex(index int) *DevicePanel {
	if wg, ok := g.Rows[index].(*DevicePanel); ok {
		return wg
	}
	return nil
}

func (g *Grid) DevicePanelByIndexPreviousVisible(index int) int {
	if index == 0 {
		return 1
	}
	for x := index; x > 1; x-- {
		if g.Rows[x].(*DevicePanel).IsVisible() {
			return x
		}
	}
	return index
}

func (g *Grid) DevicePanelByIndexNextVisible(index int) int {
	if index+1 > len(g.Rows) {
		return index
	}
	for x := index + 1; x < len(g.Rows)-1; x++ {
		if g.Rows[x].(*DevicePanel).IsVisible() {
			return x
		}
	}
	return index
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
