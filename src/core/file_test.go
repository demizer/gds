package core

import (
	"fmt"
	"testing"
)

type file struct {
	Path      string
	Size      string
	SizeBytes uint64
}

var fileTests = [...]struct {
	deviceList func() DeviceList
	fileList   func() FileList
}{
	{
		deviceList: func() DeviceList {
			var n DeviceList
			n = append(n,
				Device{Name: "Test Device 1", SizeBytes: 5368709120},
			)
			return n
		},
		fileList: func() FileList {
			var n FileList
			n = append(n,
				// File{
				// Name: "test1",
				// Path: "/mnt/data/movies/The Dark Knight - 2008.mkv",
				// Size: 28173338480,
				// },
				File{
					Name: "test1",
					Path: "../../testdata/testwalk_001/test2/alice/alice_in_wonderland_by_lewis_carroll_gutenberg.org.htm",
					Size: 668711,
				},
				File{
					Name: "test2",
					Path: "../../testdata/testwalk_001/test2/ulysses/ulysses_by_james_joyce_gutenberg.org.htm",
					Size: 1812584,
				},
			)
			return n
		},
	},
}

func TestFileSortDest(t *testing.T) {
	for _, y := range fileTests {
		f := y.fileList()
		d := y.deviceList()
		fmt.Print("\033[?25l")
		err := Sync(f, d, []string{"/dev/null"})
		fmt.Print("\033[?25h")
		if err != nil {
			t.Error(err)
		}
	}
}
