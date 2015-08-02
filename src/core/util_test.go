package core

import (
	"syscall"
	"testing"
	"time"
)

func TestSha1Sum(t *testing.T) {
	_, err := sha1sum("/root")
	if err == nil {
		t.Error("Expect: Error Got: No errors")
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
