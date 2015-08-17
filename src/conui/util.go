package conui

import "github.com/mattn/go-runewidth"

// trimStr2Runes truncates labels or strings that are to long to display in the area provided.
func trimStr2Runes(s string, w int) []rune {
	if w <= 0 {
		return []rune{}
	}
	sw := runewidth.StringWidth(s)
	if sw > w {
		return []rune(runewidth.Truncate(s, w, "..."))
	}
	return []rune(s)
}
