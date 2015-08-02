package core

import "testing"

func TestDeviceByName(t *testing.T) {
	a := &DeviceList{
		Device{Name: "Device 1"},
		Device{Name: "Device 2"},
	}
	b := a.DeviceByName("Device 1")
	if b != nil {
		t.Errorf("Got: %#v Expect: %#v\n", b, Device{Name: "Device 1"})
	}
	c := a.DeviceByName("Device 3")
	if c != nil {
		t.Errorf("Got: %#v Expect: nil\n", b)
	}

}
