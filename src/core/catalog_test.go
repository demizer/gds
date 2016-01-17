package core

import (
	"reflect"
	"testing"

	"github.com/Sirupsen/logrus"
)

// TestCatalogFileSplitAcrossDevicesNotEnoughSpace tests NewCatalog() to make sure it throws an error when there is not
// enough space to do the backup. When files are split across devices, the parent directories are duplicated. This leads to
// increased backup size causing an unexpected write error to occurr.
func TestCatalogFileSplitAcrossDevicesNotEnoughSpace(t *testing.T) {
	f := &syncTest{
		backupPath: "../../testdata/filesync_freebooks",
		deviceList: func() DeviceList {
			return DeviceList{
				&Device{
					Name:       "Test Device 0",
					SizeTotal:  1493583,
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
				},
				&Device{
					Name:       "Test Device 1",
					SizeTotal:  970000, // Needs 1009173 (including 891 bytes for the context file)
					MountPoint: NewMountPoint(t, testTempDir, "mountpoint-1-"),
				},
			}
		},
		expectErrors: func() []error {
			return []error{CatalogNotEnoughDevicePoolSpaceError{}}
		},
	}
	c := NewContext(f.backupPath, f.outputStreams, nil, f.deviceList(), f.paddingPercentage)
	var err error
	c.Files, err = NewFileList(c)
	if err != nil {
		t.Errorf("EXPECT: No errors from NewFileList() GOT: %s", err)
	}
	c.Catalog, err = NewCatalog(c)
	if err == nil || reflect.TypeOf(err) != reflect.TypeOf(f.expectErrors()[0]) {
		if err == nil {
			t.Error("EXPECT: Error TypeOf CatalogNotEnoughDevicePoolSpaceError GOT: nil")
		} else {
			t.Errorf("EXPECT: Error TypeOf CatalogNotEnoughDevicePoolSpaceError GOT: %T %q", err, err)
		}
	}
	Log.WithFields(logrus.Fields{
		"deviceName": c.Devices[0].Name,
		"mountPoint": c.Devices[0].MountPoint}).Print("Test mountpoint")
	Log.WithFields(logrus.Fields{
		"deviceName": c.Devices[1].Name,
		"mountPoint": c.Devices[1].MountPoint}).Print("Test mountpoint")
}
