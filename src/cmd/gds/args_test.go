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
	expect := path.Join(tmp0, "config.yml")
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
	expect := path.Join(tmp0, "imaginary", "config.yml")
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, "config.yml"))
	}
}

func TestGetConfigFilepathWithEnvVariable(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "$HOME")
	p, err := getConfigFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, os.Getenv("HOME"), "config.yml")
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, "config.yml"))
	}
}

func TestGetConfigFilepathWithTilde(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "~")
	p, err := getConfigFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, "/home", "config.yml")
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, "config.yml"))
	}
}

func TestGetConfigFilepathWithFullTilde(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "~/", "test")
	p, err := getConfigFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, os.Getenv("HOME"), "test", "config.yml")
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, "config.yml"))
	}
}

func TestGetConfigFilepathFull(t *testing.T) {
	tmp0, _ := ioutil.TempDir(testTempDir, "get-conf-dir-test")
	a := path.Join(tmp0, "~test", "config.yaml")
	p, err := getConfigFile(a)
	if err != nil {
		t.Errorf("EXPECT: No errors  GOT: %s", err)
	}
	expect := path.Join(tmp0, "home", "test", "config.yaml")
	if p != expect {
		t.Errorf("EXPECT: %q  GOT: %s", expect, p)
	}
	if _, err := os.Lstat(p); err != nil {
		t.Errorf("EXPECT: Path %q exists  GOT: Does not exist", path.Join(a, "config.yml"))
	}
}
