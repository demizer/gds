package core

import (
	"fmt"
	"io/ioutil"
	"testing"
)

type file struct {
	Path      string
	Size      string
	SizeBytes uint64
}

var fileTests = [...]struct {
	testName      string
	outputStreams int // Must be at least 1
	deviceList    func() DeviceList
	fileList      func() FileList
	catalog       func() Catalog
}{
	{
		testName:      "Test #1 - Simple Copy",
		outputStreams: 1,
		deviceList: func() DeviceList {
			var n DeviceList
			tmp0, _ := ioutil.TempDir("", "gds-filetests-")
			n = append(n,
				Device{
					Name:       "Test Device 0",
					SizeBytes:  28173338480,
					MountPoint: tmp0,
				},
			)
			return n
		},
		fileList: func() FileList {
			var n FileList
			n = append(n,
				File{
					Name:    "alice_in_wonderland",
					Path:    "../../testdata/testwalk_001/test2/alice/alice_in_wonderland_by_lewis_carroll_gutenberg.org.htm",
					Size:    668711,
					SrcSha1: "08cdd7178a20032c27d152a1f440334ee5f132a0",
				},
				File{
					Name:    "ulysses",
					Path:    "../../testdata/testwalk_001/test2/ulysses/ulysses_by_james_joyce_gutenberg.org.htm",
					Size:    1812584,
					SrcSha1: "d1f59d0fa64815a5f0c7527b5b4ac5c1b5a85ffb",
				},
			)
			return n
		},
	},
}

func TestFileSync(t *testing.T) {
	for _, y := range fileTests {
		c := NewContext()
		fmt.Println("Test:", y.testName, "- START")
		c.Files = y.fileList()
		c.Devices = y.deviceList()
		c.OutputStreamNum = y.outputStreams
		c.Catalog = NewCatalog(c.Devices, &c.Files)
		err := Sync(c)
		if len(err) != 0 {
			for _, e := range err {
				t.Errorf("Test: %q\n\t Error: %q\n", y.testName, e.Error())
			}
		}
		for _, cv := range c.Catalog {
			for _, cvf := range cv {
				sum, err := sha1sum(cvf.DestPath)
				if err != nil {
					t.Errorf("Test: %q\n\t Error: %q\n", y.testName, err.Error())
				}
				if cvf.SrcSha1 != sum {
					t.Errorf("Test: %q\n\t Error: %q\n", y.testName,
						fmt.Errorf("SrcSha1: %q, DestSha1: %q", cvf.SrcSha1, sum))
				}
			}
		}
		fmt.Println("Test:", y.testName, "- PASS")
	}
}
