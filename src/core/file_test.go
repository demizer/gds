package core

import "testing"

type file struct {
	Path      string
	Size      string
	SizeBytes uint64
}

var fileTests = [...]struct {
	destPath   string
	deviceList func() DeviceList
	fileList   func() FileList
}{
	{
		destPath: "/dev/null",
		deviceList: func() DeviceList {
			var n DeviceList
			n = append(n,
				Device{Name: "Test Device 1", SizeBytes: 5368709120},
				Device{Name: "Test Device 2", SizeBytes: 5368709120},
				Device{Name: "Test Device 3", SizeBytes: 5368709120},
			)
			return n
		},
		fileList: func() FileList {
			var n FileList
			n = append(n,
				File{
					Name: "test1",
					Path: "/tmp/testFiles/test1",
					Size: 1073741824,
				},
				File{
					Name: "test2",
					Path: "/tmp/testFiles/test2",
					Size: 4294967296,
				},
				File{
					Name: "test3",
					Path: "/tmp/test3",
					Size: 2147483648,
				},
			)
			return n
		},
	},
}

func TestFileSortDest(t *testing.T) {
	for _, y := range fileTests {
		q := y.fileList()
		z := y.deviceList()
		q.Sync(z)
	}
}
