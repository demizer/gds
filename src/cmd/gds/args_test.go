package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/Sirupsen/logrus"
)

var testTempDir = "../../../testdata/temp/"

func init() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Enable debug output.")
	flag.Parse()
	if debug {
		log.Out = os.Stdout
		log.Level = logrus.DebugLevel
	}
}

func TestGetConfigFileDirExists(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	p, err := getConfigFile(tmp0)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, GDS_CONFIG_NAME)
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
}

func TestGetConfigFileDirDoesNotExist(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "imaginary")
	p, err := getConfigFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, "imaginary", GDS_CONFIG_NAME)
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, GDS_CONFIG_NAME))
	}
}

func TestGetConfigFileDirDoesNotExistDiffFileExt(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "imaginary", "config.myext")
	p, err := getConfigFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, "imaginary", "config.myext")
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", a)
	}
}

func TestGetConfigFilepathWithEnvVariable(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "$HOME")
	p, err := getConfigFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, os.Getenv("HOME"), GDS_CONFIG_NAME)
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, GDS_CONFIG_NAME))
	}
}

func TestGetConfigFilepathWithTilde(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "~")
	p, err := getConfigFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, "/home", GDS_CONFIG_NAME)
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, GDS_CONFIG_NAME))
	}
}

func TestGetConfigFilepathWithFullTilde(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "~/", "test")
	p, err := getConfigFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, os.Getenv("HOME"), "test", GDS_CONFIG_NAME)
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, GDS_CONFIG_NAME))
	}
}

func TestGetConfigFilepathFull(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "~test")
	p, err := getConfigFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, "home", "test", GDS_CONFIG_NAME)
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, GDS_CONFIG_NAME))
	}
}

func TestGetContextFilepathEnvironVariableFull(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "$GDS_CONFIG_DIR")
	os.Setenv("GDS_CONFIG_DIR", "test")
	p, err := getContextFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, "test", GDS_CONTEXT_FILENAME)
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, GDS_CONTEXT_FILENAME))
	}
}

func TestGetContextFilepathFull(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "~test")
	p, err := getContextFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, "home", "test", GDS_CONTEXT_FILENAME)
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, GDS_CONTEXT_FILENAME))
	}
}

func TestGetContextFilepathFullDiffName(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "~test", "config.json")
	p, err := getContextFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, "home", "test", "config.json")
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, "config.json"))
	}
}
