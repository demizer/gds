package core

import (
	"io/ioutil"
	"testing"
)

func TestContextFromBytes(t *testing.T) {
	b, err := ioutil.ReadFile("../../testdata/context/config.yml")
	if err != nil {
		t.Error(err)
	}
	ctx, err := ContextFromBytes(b)
	if err != nil {
		t.Error(err)
	}
	if len(ctx.Devices) != 2 {
		t.Errorf("Expect: 2 devices Got: %d", len(ctx.Devices))
	}
}

func TestContextFromBytesNoDevices(t *testing.T) {
	a := []byte(`
backupPath: "/home"
outputStreams: 1`)
	_, err := ContextFromBytes(a)
	if err == nil {
		t.Errorf("Expect: DeviceNotFoundError Got: %q", err.Error())
	}
	if b, ok := err.(*ContextFileHasNoDevicesError); !ok {
		t.Errorf("Expect: %T Got: %T", new(ContextFileHasNoDevicesError), b)
	}
	if new(ContextFileHasNoDevicesError).Error() == "" {
		t.Error("Missing error message")
	}
}

func TestContextFromBytesBadYaml(t *testing.T) {
	a := []byte("backupPath: [")
	_, err := ContextFromBytes(a)
	if err == nil {
		t.Error("Expect: Error Got: nil")
	}
}

func TestContextFromPath(t *testing.T) {
	a, err := ContextFromPath("no config file")
	if err == nil {
		t.Errorf("Expect: Error Got: %#v", a)
	}
	b, err := ContextFromPath("../../testdata/context/config.yml")
	if err != nil {
		t.Errorf("Expect: Context Got: %q", err.Error())
	}
	if len(b.Devices) != 2 {
		t.Errorf("Expect: 2 devices Got: %d", len(b.Devices))
	}
}
