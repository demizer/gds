package conui

import (
	"flag"
	"os"

	"github.com/Sirupsen/logrus"
)

func init() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Enable debug output.")
	flag.Parse()
	if debug {
		Log.Out = os.Stdout
		Log.Level = logrus.DebugLevel
	}
}
