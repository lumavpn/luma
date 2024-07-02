package util

import "context"

func Must(errs ...error) {
	for _, err := range errs {
		if err != nil {
			panic(err)
		}
	}
}

func Must1[T any](result T, err error) T {
	if err != nil {
		panic(err)
	}
	return result
}

func Done(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
