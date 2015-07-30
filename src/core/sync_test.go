package core

import "testing"

func TestSyncNoCatalog(t *testing.T) {
	c := NewContext("../../testdata/filesync_freebooks")
	c.Files, _ = NewFileList(c)
	errs := Sync(c)
	expectErr := "No catalog data, nothing to do."
	var found bool
	for _, e := range errs {
		if e.Error() == expectErr {
			found = true
		}
	}
	if !found {
		t.Errorf("Expect: %q\n\t Got: %s", expectErr, spd.Sprint(errs))
	}
}
