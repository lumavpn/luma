package log

import (
	"fmt"
	"time"

	"github.com/lumavpn/luma/common/observable"
)

var (
	logCh  = make(chan Event)
	source = observable.NewObservable[Event](logCh)
)

type Event struct {
	Level   LogLevel  `json:"level"`
	Message string    `json:"msg"`
	Time    time.Time `json:"time"`
}

func newEvent(level LogLevel, format string, args ...any) *Event {
	event := Event{
		Level:   level,
		Time:    time.Now(),
		Message: fmt.Sprintf(format, args...),
	}
	logCh <- event

	return &event
}

func Subscribe() (observable.Subscription[Event], error) {
	return source.Subscribe()
}

func UnSubscribe(sub observable.Subscription[Event]) {
	source.UnSubscribe(sub)
}
