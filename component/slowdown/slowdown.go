package slowdown

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/jpillora/backoff"
)

type SlowDown struct {
	errTimes atomic.Int64
	backoff  *backoff.Backoff
}

func New() *SlowDown {
	return &SlowDown{
		backoff: &backoff.Backoff{
			Min:    10 * time.Millisecond,
			Max:    1 * time.Second,
			Factor: 2,
			Jitter: true,
		},
	}
}

func (s *SlowDown) Wait(ctx context.Context) (err error) {
	timer := time.NewTimer(s.backoff.Duration())
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
		err = ctx.Err()
	}
	return
}

func Do[T any](s *SlowDown, ctx context.Context, fn func() (T, error)) (t T, err error) {
	if s.errTimes.Load() > 10 {
		err = s.Wait(ctx)
		if err != nil {
			return
		}
	}
	t, err = fn()
	if err != nil {
		s.errTimes.Add(1)
		return
	}
	s.errTimes.Store(0)
	s.backoff.Reset()
	return
}
