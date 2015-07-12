package main

import "testing"

// Test cases
// 1. A file is too big for all drives
// 2. A file is too big for any one drive, but can fit on all backup drives
// 3. A symlink on one drive that refers to a files saved on another drive due
//    to space limitations.

type file struct {
	Path      string
	Size      string
	SizeBytes uint64
}

var fileTests = [...]struct {
	driveList func() *DriveList
	fileList  func() *FileList
}{
	{
		driveList: func() *DriveList {
			n := new(DriveList)
			*n = append(*n,
				Drive{Name: "Test Drive 1", SizeBytes: 5368709120},
				Drive{Name: "Test Drive 2", SizeBytes: 5368709120},
				Drive{Name: "Test Drive 3", SizeBytes: 5368709120},
			)
			return n
		},
		fileList: func() *FileList {
			n := new(FileList)
			*n = append(*n,
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
	// for _, y := range fileTests {
	// z := y.driveList()
	// q := y.fileList()
	// }
}
