package log

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"go.uber.org/atomic"
)

// _defaultLevel is the package default logging level.
var _defaultLevel = atomic.NewUint32(uint32(DebugLevel))

func init() {
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05.999999999Z07:00",
	})
}

func SetLevel(level LogLevel) {
	_defaultLevel.Store(uint32(level))
}

// SetOutput sets the standard logger output.
func SetOutput(out io.Writer) {
	logrus.SetOutput(out)
}

func Level() LogLevel {
	return LogLevel(_defaultLevel.Load())
}

func Debug(message string) {
	logf(DebugLevel, message)
}

func Debugf(format string, args ...any) {
	logf(DebugLevel, format, args...)
}

func Info(message string) {
	logf(InfoLevel, message)
}

func Infof(format string, args ...any) {
	logf(InfoLevel, format, args...)
}

func Warn(message string) {
	logf(WarnLevel, message)
}

func Warnf(format string, args ...any) {
	logf(WarnLevel, format, args...)
}

func Error(err error) {
	if err != nil {
		logf(ErrorLevel, err.Error())
	}
}

func Errorf(format string, args ...any) {
	logf(ErrorLevel, format, args...)
}

func Fatal(err any) {
	logrus.Fatal(err)
}

func Fatalf(format string, args ...any) {
	logrus.Fatalf(format, args...)
}

func logf(level LogLevel, format string, args ...any) {
	event := newEvent(level, format, args...)
	if uint32(event.Level) > _defaultLevel.Load() {
		return
	}

	switch level {
	case DebugLevel:
		logrus.WithTime(event.Time).Debugln(event.Message)
	case InfoLevel:
		logrus.WithTime(event.Time).Infoln(event.Message)
	case WarnLevel:
		logrus.WithTime(event.Time).Warnln(event.Message)
	case ErrorLevel:
		logrus.WithTime(event.Time).Errorln(event.Message)
	}
}
