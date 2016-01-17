package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

var spd = spew.ConfigState{Indent: "\t"} //, DisableMethods: true}

var devices = []string{"/mnt/backup1", "/mnt/backup2"}

func main() {
	type Dev struct {
		mount string
		dev   string
		uuid  string
	}
	devs := make(map[string]*Dev)
	f, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	// fmt.Println(string(f))
	for _, v := range strings.Split(string(f), "\n") {
		for _, x := range devices {
			if strings.Contains(v, x) {
				// fmt.Println(v)
				devName := strings.Split(v, " ")[0]
				mnt := strings.Split(v, " ")[1]
				devs[mnt] = &Dev{dev: devName}
			}
		}
		// fmt.Println(strings.Contains(string(v))
	}
	wf := func(p string, i os.FileInfo, err error) error {
		if p == "/dev/disk/by-uuid/" {
			return err
		}
		// fmt.Println(p)
		for x, y := range devs {
			tgt, err := os.Readlink(p)
			if err != nil {
				fmt.Println("ERROR:", err)
				os.Exit(1)
			}
			if filepath.Base(y.dev) == filepath.Base(tgt) {
				fmt.Println(i.Name())
				devs[x].uuid = i.Name()
			}
		}
		return err
	}
	err = filepath.Walk("/dev/disk/by-uuid/", wf)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
}
