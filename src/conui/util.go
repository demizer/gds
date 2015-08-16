package conui

import (
	"os"

	"github.com/mattn/go-runewidth"
)

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

// Used to print debug information to a file. Only temporary.
//
// DO NOT COMMIT
//
func writeString(str string) {
	oFile, err := os.OpenFile("/tmp/debug", os.O_APPEND|os.O_WRONLY, 0644)
	defer oFile.Close()
	if err != nil {
		panic(err)
	}
	_, err = oFile.WriteString(str)
	if err != nil {
		panic(err)
	}
}
