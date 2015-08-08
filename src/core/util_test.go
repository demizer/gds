package core

import (
	"io/ioutil"
	"syscall"
	"testing"
	"time"
)

// NewMountPoint creates a new test mountpoint and returns a string with the directory ready for usage. Any errors creating
// the mountpoint wil Fail the test.
func NewMountPoint(t *testing.T, basePath string, prefix string) string {
	p, err := ioutil.TempDir(testTempDir, prefix)
	if err != nil {
		t.Fatalf("EXPECT: path to temp mount GOT: %s", err)
	}
	return p
}

func TestSha1Sum(t *testing.T) {
	_, err := sha1sum("/root")
	if err == nil {
		t.Error("EXPECT: error permission denied GOT: No errors")
	}
}

func TestLUtimesNano(t *testing.T) {
	f := File{
		ModTime: time.Now(),
	}
	mTimeval := syscall.NsecToTimespec(f.ModTime.UnixNano())
	times := []syscall.Timespec{
		mTimeval,
		mTimeval,
	}
	err := LUtimesNano("/root", times)
	if err == nil {
		t.Errorf("Expect: Error Got: %q", err)
	}
	err = LUtimesNano("/root\x00", times)
	if err == nil {
		t.Errorf("Expect: Error Got: %q", err)
	}
}
