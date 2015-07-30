package core

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/davecgh/go-spew/spew"
)

var spd = spew.ConfigState{Indent: "\t"} //, DisableMethods: true}

var Log = &logrus.Logger{
	Out:       os.Stdout,
	Formatter: new(TextFormatter),
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.InfoLevel,
}
