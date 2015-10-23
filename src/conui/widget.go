// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Copyright 2015 Jesus Alvarez <jeezusjr@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package conui

type Widget interface {
	Bufferer
	IsSelected() bool
	SetSelected(bool)
	IsVisible() bool
	SetVisible(bool)
	Prompt() *PromptAction
	SetPrompt(*PromptAction)
	Width() int
	SetWidth(int)
	Height() int
	SetHeight(int)
	X() int
	SetX(int)
	Y() int
	SetY(int)
}
