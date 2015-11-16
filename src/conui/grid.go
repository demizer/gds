// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Copyright 2015 Jesus Alvarez <jeezusjr@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package conui

type Grid struct {
	SelectedDevicePanel int
	DevicePanels        []Widget
	DevicePanelHeight   int
	ProgressPanel       *ProgressGauge
	ProgressPanelHeight int
	Width               int
	X                   int
	Y                   int
	BgColor             Attribute
}

func NewGrid() *Grid {
	return &Grid{DevicePanels: []Widget{}, ProgressPanel: &ProgressGauge{}}
}

func (g *Grid) selected() int {
	var x int
	var y Widget
	for x, y = range g.DevicePanels {
		if y.IsSelected() {
			break
		}
	}
	return x
}

func (g *Grid) deselectAll() {
	for x, _ := range g.DevicePanels {
		if _, ok := g.DevicePanels[x].(*DevicePanel); !ok {
			continue
		}
		g.DevicePanels[x].SetSelected(false)
		g.DevicePanels[x].(*DevicePanel).Border.FgColor = ColorWhite
	}
}

func (g *Grid) NumVisible() int {
	num := 0
	for x := 0; x < len(g.DevicePanels); x++ {
		if g.DevicePanels[x].(*DevicePanel).IsVisible() {
			num += 1
		}
	}
	return num
}

func (g *Grid) Select(index int) *DevicePanel {
	g.deselectAll()
	wg := g.DevicePanels[index].(*DevicePanel)
	g.SelectedDevicePanel = index
	wg.SetVisible(true)
	return wg
}

func (g *Grid) scrollVisible() {
	visibleArea := TermHeight() - 5

	Log.Debugf("scrollVisible: visibleArea: %d g.SelectedDevicePanel: %d numVisibleWidgets: %d",
		visibleArea, g.SelectedDevicePanel, g.NumVisible())

	if (TermHeight() - g.ProgressPanelHeight) < g.DevicePanelHeight {
		Log.Debugln("scrollVisible: Rendering only selected panel (not enough term height)")
		for x := 0; x < len(g.DevicePanels); x++ {
			if dp := g.DevicePanelByIndex(x); dp != nil {
				if x == g.SelectedDevicePanel {
					dp.SetY(g.ProgressPanelHeight)
					continue
				}
			}
		}
	} else if g.SelectedDevicePanel == 0 {
		Log.Debugln("scrollVisible: First visible panel selected")
		yPos := g.ProgressPanelHeight
		for x := 0; x < len(g.DevicePanels); x++ {
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
		for x := len(g.DevicePanels) - 1; x >= 0; x-- {
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
		for x := 0; x < len(g.DevicePanels); x++ {
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
					Log.Debugf("scrollVisible: g.DevicePanels[%d].Y = %d", visiblePanels[x], ny)
					dp.SetY(ny)
					continue
				} else if x == selRowInVis {
					Log.Debugf("scrollVisible: g.DevicePanels[%d].Y = %d", g.SelectedDevicePanel, yPos)
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
			g.SelectedDevicePanel = len(g.DevicePanels) - 1
		}
		if wg, ok = g.DevicePanels[g.SelectedDevicePanel].(*DevicePanel); ok {
			visible := wg.IsVisible()
			Log.Debugf("SELECTED previous Widget[%d].Visible = %t", g.SelectedDevicePanel, visible)
			if visible {
				Log.Debugf("SelectPrevious: termHeight: %d wg.Y: %d wg.Height: %d", TermHeight(), wg.Y(), wg.Height())
				if (g.NumVisible() * g.DevicePanelHeight) > (TermHeight() - 5) {
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
		if g.SelectedDevicePanel > len(g.DevicePanels)-1 {
			g.SelectedDevicePanel = 0
		}
		if wg, ok = g.DevicePanels[g.SelectedDevicePanel].(*DevicePanel); ok {
			visible := wg.IsVisible()
			Log.Debugf("SelectNext: Widget[%d].Visible = %t", g.SelectedDevicePanel, visible)
			if visible {
				Log.Debugf("SelectNext: termHeight: %d wg.Y: %d wg.Height: %d", TermHeight(), wg.Y(), wg.Height())
				if (g.NumVisible() * g.DevicePanelHeight) > (TermHeight() - 5) {
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
	return g.DevicePanels[g.SelectedDevicePanel].(*DevicePanel)
}

func (g *Grid) DevicePanelByIndex(index int) *DevicePanel {
	if wg, ok := g.DevicePanels[index].(*DevicePanel); ok {
		return wg
	}
	return nil
}

func (g *Grid) DevicePanelByIndexPreviousVisible(index int) int {
	if index == 0 {
		return 0
	}
	for x := index; x >= 0; x-- {
		if g.DevicePanels[x].(*DevicePanel).IsVisible() {
			return x
		}
	}
	return index
}

func (g *Grid) DevicePanelByIndexNextVisible(index int) int {
	if index+1 > len(g.DevicePanels) {
		return index
	}
	for x := index + 1; x < len(g.DevicePanels); x++ {
		if g.DevicePanels[x].(*DevicePanel).IsVisible() {
			return x
		}
	}
	return index
}

func (g *Grid) PromptByIndex(index int, prompt PromptAction) *DevicePanel {
	wg := g.DevicePanels[index].(*DevicePanel)
	if wg != nil {
		g.Select(index)
		wg.SetSelected(true)
		wg.SetVisible(true)
	}
	return wg
}

// Buffer implments Bufferer interface.
func (g Grid) Buffer() []Point {
	ps := []Point{}
	// LIFO with the rows... row 0 has the highest priority, meaning the codes in row 0 cannot be overwritten.
	for x := len(g.DevicePanels) - 1; x > 0; x-- {
		ps = append(ps, g.DevicePanels[x].Buffer()...)
	}
	return ps
}
