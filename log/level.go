package log

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type LogLevel uint32

const (
	SilentLevel LogLevel = iota
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)

var (
	// LogLevelMapping is a mapping for LogLevel enum
	LogLevelMapping = map[string]LogLevel{
		ErrorLevel.String():  ErrorLevel,
		WarnLevel.String():   WarnLevel,
		InfoLevel.String():   InfoLevel,
		DebugLevel.String():  DebugLevel,
		SilentLevel.String(): SilentLevel,
	}
)

// UnmarshalJSON deserialize LogLevel with json
func (l *LogLevel) UnmarshalJSON(data []byte) error {
	var lvl string
	if err := json.Unmarshal(data, &lvl); err != nil {
		return err
	}

	level, exist := LogLevelMapping[lvl]
	if !exist {
		return errors.New("invalid log level")
	}
	*l = level
	return nil
}

// MarshalJSON serialize LogLevel with json
func (l LogLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

// UnmarshalYAML unserialize LogLevel with yaml
func (e *LogLevel) UnmarshalYAML(unmarshal func(any) error) error {
	var tp string
	if err := unmarshal(&tp); err != nil {
		return err
	}
	mode, exist := LogLevelMapping[strings.ToLower(tp)]
	if !exist {
		return errors.New("invalid log level")
	}
	*e = mode
	return nil
}

// MarshalJSON serialize LogLevel with yaml
func (l LogLevel) MarshalYAML() (any, error) {
	return l.String(), nil
}

func (l LogLevel) String() string {
	switch l {
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	case DebugLevel:
		return "debug"
	case SilentLevel:
		return "silent"
	default:
		return "unknown"
	}
}

func ParseLevel(l string) (LogLevel, error) {
	if lvl, ok := LogLevelMapping[strings.ToLower(l)]; ok {
		return lvl, nil
	}
	return LogLevel(0), fmt.Errorf("not a valid log level: %q", l)
}
