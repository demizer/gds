package core

import "testing"

func TestDeviceByName(t *testing.T) {
	a := &DeviceList{
		&Device{Name: "Device 1"},
		&Device{Name: "Device 2"},
	}
	_, err := a.DeviceByName("Device 1")
	if err != nil {
		t.Errorf("Expect: Device 1 Got: %q", err.Error())
	}
	_, err = a.DeviceByName("Device 3")
	if d, ok := err.(*DeviceNotFoundError); !ok {
		t.Errorf("Expect: %T Got: %T", new(DeviceNotFoundError), d)
	}
	if new(DeviceNotFoundError).Error() == "" {
		t.Error("Missing error message")
	}
}
