package watchdog

import (
	"context"
)

type WatchDog interface {
	Run(ctx context.Context) error
	ResetCountdown()
}
