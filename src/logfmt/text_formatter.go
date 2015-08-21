package logfmt

import (
	"bytes"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
)

const (
	nocolor = 0
	red     = 31
	green   = 32
	yellow  = 33
	blue    = 34
	magenta = 35
	gray    = 37
)

var (
	baseTimestamp time.Time
	isTerminal    bool
)

func init() {
	baseTimestamp = time.Now()
	isTerminal = logrus.IsTerminal()
}

func miniTS() int {
	return int(time.Since(baseTimestamp) / time.Second)
}

// TextFormatter satisfies the logrus formatter interface.
type TextFormatter struct {
	// Set to true to bypass checking for a TTY before outputting colors.
	ForceColors bool

	// Force disabling colors.
	DisableColors bool

	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool

	// Enable logging the full timestamp when a TTY is attached instead of just
	// the time passed since beginning of execution.
	FullTimestamp bool

	// TimestampFormat to use for display when a full timestamp is printed
	TimestampFormat string

	// The fields are sorted by default for a consistent output. For applications
	// that log extremely frequently and don't use the JSON formatter this may not
	// be desired.
	DisableSorting bool
}

// Format formats a log entry into pretty color output, or key=value formatted output in case of output being sent to a file.
func (f *TextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var err error
	var keys []string = make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}

	if !f.DisableSorting {
		sort.Strings(keys)
	}

	b := &bytes.Buffer{}

	prefixFieldClashes(entry.Data)

	isColorTerminal := isTerminal && (runtime.GOOS != "windows")
	isColored := (f.ForceColors || isColorTerminal) && !f.DisableColors

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = logrus.DefaultTimestampFormat
	}
	if isColored {
		f.printColored(b, entry, keys, timestampFormat)
	} else {
		var aErr, kErr error
		if !f.DisableTimestamp {
			aErr = f.appendKeyValue(b, "time", entry.Time.Format(timestampFormat))
		}
		aErr2 := f.appendKeyValue(b, "level", entry.Level.String())
		aErr3 := f.appendKeyValue(b, "msg", entry.Message)
		for _, key := range keys {
			kErr = f.appendKeyValue(b, key, entry.Data[key])
			if err != nil {
				break
			}
		}
		if aErr != nil || aErr2 != nil || aErr3 != nil || kErr != nil {
			return nil, err
		}
	}

	err = b.WriteByte('\n')
	return b.Bytes(), err
}

func (f *TextFormatter) printColored(b *bytes.Buffer, entry *logrus.Entry, keys []string, timestampFormat string) {
	var levelColor int
	switch entry.Level {
	case logrus.DebugLevel:
		levelColor = blue
	case logrus.WarnLevel:
		levelColor = magenta
	case logrus.ErrorLevel:
		levelColor = yellow
	case logrus.FatalLevel, logrus.PanicLevel:
		levelColor = red
	default:
		levelColor = green
	}

	// levelText := strings.ToUpper(entry.Level.String())[0:4]
	levelText := strings.ToUpper(entry.Level.String())
	padd := strings.Repeat(" ", 7-len(levelText))

	if !f.FullTimestamp && !f.DisableTimestamp {
		fmt.Fprintf(b, fmt.Sprintf("[%%04d] \x1b[1m\x1b[%%dm%%s\x1b[0m %s %%-24s ", padd),
			miniTS(), levelColor, levelText, entry.Message)
	} else if !f.FullTimestamp && f.DisableTimestamp {
		fmt.Fprintf(b, fmt.Sprintf("\x1b[1m\x1b[%%dm%%s\x1b[0m %s %%-24s ", padd), levelColor, levelText,
			entry.Message)
	} else {
		fmt.Fprintf(b, fmt.Sprintf("[%%s] \x1b[1m\x1b[%%dm%%s\x1b[0m %s %%-24s ", padd),
			entry.Time.Format(timestampFormat), levelColor, levelText, entry.Message)
	}
	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, " \x1b[1m\x1b[%dm%s\x1b[0m=%+v", levelColor, k, v)
	}
}

func needsQuoting(text string) bool {
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.') {
			return false
		}
	}
	return true
}

func (f *TextFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) error {

	_, err := b.WriteString(key)
	if err != nil {
		return err
	}
	err = b.WriteByte('=')
	if err != nil {
		return err
	}

	switch value := value.(type) {
	case string:
		if needsQuoting(value) {
			_, err := b.WriteString(value)
			if err != nil {
				return err
			}
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	case error:
		errmsg := value.Error()
		if needsQuoting(errmsg) {
			_, err := b.WriteString(errmsg)
			if err != nil {
				return err
			}
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	default:
		fmt.Fprint(b, value)
	}

	err = b.WriteByte(' ')
	return err
}

// This is to not silently overwrite `time`, `msg` and `level` fields when
// dumping it. If this code wasn't there doing:
//
//  logrus.WithField("level", 1).Info("hello")
//
// Would just silently drop the user provided level. Instead with this code
// it'll logged as:
//
//  {"level": "info", "fields.level": 1, "msg": "hello", "time": "..."}
//
// It's not exported because it's still using Data in an opinionated way. It's to
// avoid code duplication between the two default formatters.
func prefixFieldClashes(data logrus.Fields) {
	_, ok := data["time"]
	if ok {
		data["fields.time"] = data["time"]
	}

	_, ok = data["msg"]
	if ok {
		data["fields.msg"] = data["msg"]
	}

	_, ok = data["level"]
	if ok {
		data["fields.level"] = data["level"]
	}
}
