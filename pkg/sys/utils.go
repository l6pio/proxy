package sys

import (
	"context"
	"time"
)

var ExitAction []func()

func AddExitAction(fn func()) {
	ExitAction = append(ExitAction, fn)
}

func WaitUntilTimeout(timeout time.Duration, fn func() bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	timer := time.NewTimer(time.Second)
	select {
	case <-timer.C:
		if fn() {
			return
		}
	case <-ctx.Done():
	}
}
