package core

// import (
// "encoding/json"
// "io/ioutil"
// "testing"
// )

// func TestContextFromBytes(t *testing.T) {
// b, err := ioutil.ReadFile("../../testdata/context/config.yml")
// if err != nil {
// t.Error(err)
// }
// ctx, err := NewContextFromYaml(b)
// if err != nil {
// t.Error(err)
// }
// if len(ctx.Devices) != 2 {
// t.Errorf("Expect: 2 devices Got: %d", len(ctx.Devices))
// }
// }

// func TestContextFromBytesNoDevices(t *testing.T) {
// a := []byte(`
// backupPath: "/home"
// outputStreams: 1`)
// _, err := NewContextFromYaml(a)
// if err == nil {
// t.Errorf("Expect: DeviceNotFoundError Got: %q", err.Error())
// }
// if b, ok := err.(*ContextFileHasNoDevicesError); !ok {
// t.Errorf("Expect: %T Got: %T", new(ContextFileHasNoDevicesError), b)
// }
// if new(ContextFileHasNoDevicesError).Error() == "" {
// t.Error("Missing error message")
// }
// }

// func TestContextFromBytesBadYaml(t *testing.T) {
// a := []byte("backupPath: [")
// _, err := NewContextFromYaml(a)
// if err == nil {
// t.Error("Expect: Error Got: nil")
// }
// }

// func TestContextFromPath(t *testing.T) {
// a, err := ContextFromPath("no config file")
// if err == nil {
// t.Errorf("Expect: Error Got: %#v", a)
// }
// b, err := ContextFromPath("../../testdata/context/config.yml")
// if err != nil {
// t.Errorf("Expect: Context Got: %q", err.Error())
// }
// if len(b.Devices) != 2 {
// t.Errorf("Expect: 2 devices Got: %d", len(b.Devices))
// }
// }

// func TestContextMarshalJSON(t *testing.T) {
// f := &syncTest{
// backupPath: "../../testdata/filesync_freebooks",
// deviceList: func() DeviceList {
// return DeviceList{
// &Device{
// Name:       "Test Device 0",
// SizeTotal:  3499350,
// MountPoint: NewMountPoint(t, testTempDir, "mountpoint-0-"),
// },
// }
// },
// }
// c := NewContext(f.backupPath, f.outputStreams, nil, f.deviceList(), f.paddingPercentage)
// var err error
// c.Files, err = NewFileList(c)
// if err != nil {
// t.Errorf("Expect: No Errors  Got: %s", err)
// }
// c.Catalog, err = NewCatalog(c)
// if err != nil {
// t.Errorf("EXPECT: No errors from NewCatalog() GOT: %s", err)
// }
// // Turn into json
// j, err := json.Marshal(c)
// if err != nil {
// t.Errorf("Expect: No Errors  Got: %s", err)
// }
// // From json to context
// _, err = NewContextFromJSON(j)
// if err != nil {
// t.Errorf("Expect: No Errors  Got: %s", err)
// }
// // if !reflect.DeepEqual(c, c2) {
// // t.Error("Expect: Context from JSON DeepEqual = True  Got: False")
// // }
// }
