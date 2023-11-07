package sysrq

import (
	"context"
	"fmt"
	"time"
)

type WatchDog struct {
	timeout        time.Duration
	resetTimerChan chan struct{}
}

func NewWatchdog(timeout time.Duration) *WatchDog {
	resetTimerChan := make(chan struct{})
	return &WatchDog{
		resetTimerChan: resetTimerChan,
		timeout:        timeout,
	}
}

func (w WatchDog) ResetCountdown() {
	w.resetTimerChan <- struct{}{}
}

func (w WatchDog) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	timeoutInt := int(w.timeout.Seconds())
	counter := timeoutInt
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			fmt.Println(counter)
			if counter <= 0 {
				fmt.Println("booooom")
			}
			counter--
		case <-w.resetTimerChan:
			counter = timeoutInt
		}
	}
}
