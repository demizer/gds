package core

import "testing"

type file struct {
	Path      string
	Size      string
	SizeBytes uint64
}

var fileTests = [...]struct {
	destPath  string
	driveList func() DriveList
	fileList  func() FileList
}{
	{
		destPath: "/dev/null",
		driveList: func() DriveList {
			var n DriveList
			n = append(n,
				Drive{Name: "Test Drive 1", SizeBytes: 5368709120},
				Drive{Name: "Test Drive 2", SizeBytes: 5368709120},
				Drive{Name: "Test Drive 3", SizeBytes: 5368709120},
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
		z := y.driveList()
		q.Sync(z)
	}
}
