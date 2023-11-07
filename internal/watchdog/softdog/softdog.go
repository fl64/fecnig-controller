package softdog

import (
	"context"
	"fmt"
	"os"
)

type WatchDog struct {
	device         string
	resetTimerChan chan struct{}
	file           *os.File
}

func NewWatchdog(device string) *WatchDog {
	resetTimerChan := make(chan struct{})
	return &WatchDog{
		device:         device,
		resetTimerChan: resetTimerChan,
	}
}

func (w WatchDog) ResetCountdown() {
	w.resetTimerChan <- struct{}{}
}

func (w *WatchDog) Run(ctx context.Context) error {
	var err error
	w.file, err = os.OpenFile(w.device, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("Unable to open watchdog device", err)
	}
	defer w.file.Close()

	feedWatchdog := func(s string) {
		fmt.Fprint(w.file, s)
		w.file.Sync()
	}

	for {
		select {
		case <-ctx.Done():
			feedWatchdog("V")
			return nil
		case <-w.resetTimerChan:
			feedWatchdog("1")
			// как гарантировать что успеет?
		}
	}
}
