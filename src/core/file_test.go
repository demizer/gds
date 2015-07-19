package core

import "testing"

type file struct {
	Path      string
	Size      string
	SizeBytes uint64
}

var fileTests = [...]struct {
	outputStreams int
	deviceList    func() DeviceList
	fileList      func() FileList
}{
	{
		outputStreams: 2,
		deviceList: func() DeviceList {
			var n DeviceList
			n = append(n,
				Device{
					Name:       "Test Device 1",
					SizeBytes:  8122329361,
					MountPoint: "/dev/null",
				},
				Device{
					Name:       "Test Device 2",
					SizeBytes:  73019310000,
					MountPoint: "/dev/null",
				},
			)
			return n
		},
		fileList: func() FileList {
			var n FileList
			n = append(n,
				// File{
				// Name: "dark_knight",
				// Path: "/mnt/data/movies/The Dark Knight - 2008.mkv",
				// Size: 28173338480,
				// },
				File{
					Name: "The Wolf and the Lion",
					Path: "/mnt/data/shows/Game of Thrones/Season 1/Game of Thrones - 1x05 - The Wolf and the Lion.mkv",
					Size: 8122329361,
				},
				File{
					Name: "elementaryos_2013",
					Path: "/mnt/data/files/images/elementaryos-stable-amd64.20130810.iso",
					Size: 727711744,
				},
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
		c := NewContext()
		c.Files = y.fileList()
		c.Devices = y.deviceList()
		c.OutputStreamNum = y.outputStreams
		c.Catalog = c.Files.catalog(c.Devices)
		// spd.Dump(c)
		// os.Exit(1)
		err := Sync(c)
		if err != nil {
			t.Error(err)
		}
	}
}
