package log

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// Logger wraps a "github.com/go-kit/log/Logger" to provide a more convenient way
// to log with a particular level.
// Note that it implements the "github.com/go-kit/log/Logger" interface.
type Logger struct {
	l log.Logger
}

func (l Logger) Log(keyvals ...interface{}) error {
	return l.l.Log(keyvals...)
}

func (l Logger) Debug() Logger {
	l.l = level.Debug(l.l)
	return l
}

func (l Logger) Info() Logger {
	l.l = level.Info(l.l)
	return l
}

func (l Logger) Warn() Logger {
	l.l = level.Warn(l.l)
	return l
}

func (l Logger) Error() Logger {
	l.l = level.Error(l.l)
	return l
}

func (l Logger) With(keyvals ...interface{}) Logger {
	l.l = log.With(l.l, keyvals...)
	return l
}

type Option string

const AllowDebug Option = "allowDebug"
const AllowInfo Option = "allowInfo"
const AllowWarn Option = "allowWarn"
const AllowError Option = "allowError"
const PrettyPrint Option = "prettyPrint"

// Default logger that encodes log events as a single json object
// and writes them to stdout. Utc timestamps and caller (file + line) are added to each log event.
//
// There are four different log levels: error/warn/info/debug (in descending order of severity).
// Pass an option to filter out log messages below a certain level.
// E.g. if AllowInfo is passed, only log messages with level info/warn/error or without a level will be logged.
// By default only log messages with or above level info are logged.
func DefaultJSONLogger(options ...Option) *Logger {
	levelFilter := level.AllowInfo()
	prettyPrint := false
	for _, option := range options {
		switch option {
		case AllowDebug:
			levelFilter = level.AllowDebug()
		case AllowInfo:
			levelFilter = level.AllowInfo()
		case AllowWarn:
			levelFilter = level.AllowWarn()
		case AllowError:
			levelFilter = level.AllowError()
		case PrettyPrint:
			prettyPrint = true
		}
	}

	var writer io.Writer = os.Stdout
	if prettyPrint {
		writer = newPrettyJSONWriter(writer)
	}

	var logger log.Logger
	{
		//	Although not explicitly stated in the docs, os.File should be safe for concurrent use,
		//	especially since NewJSONLogger only calls w.Write at most once per log event.
		//	Otherwise we would e.g. need to wrap os.Stdout with log.NewSyncWriter.
		logger = log.NewJSONLogger(writer)
		logger = level.NewFilter(logger, levelFilter, level.SquelchNoLevel(false))
		logger = log.With(logger, "timestamp", log.DefaultTimestampUTC)
	}

	return &Logger{l: logger}
}

// Can be used to pretty print json log messages,
// e.g. useful for development/testing.
type prettyJSONWriter struct {
	w io.Writer
}

func newPrettyJSONWriter(w io.Writer) prettyJSONWriter {
	return prettyJSONWriter{w: w}
}

func (w prettyJSONWriter) Write(p []byte) (int, error) {
	var buf bytes.Buffer

	err := json.Indent(&buf, p, "", "  ")
	if err != nil {
		return w.w.Write(p)
	}
	return w.w.Write(buf.Bytes())
}
