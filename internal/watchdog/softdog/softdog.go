package softdog

import (
	"errors"
	"fmt"
	"os"
)

type WatchDog struct {
	watchdogDeviceName string
	watchdogDevice     *os.File
}

func NewWatchdog(device string) *WatchDog {
	return &WatchDog{
		watchdogDeviceName: device,
	}
}

func (w *WatchDog) Start() error {
	var err error
	w.watchdogDevice, err = os.OpenFile(w.watchdogDeviceName, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("Unable to open watchdog device %w", err)
	}
	return nil
}

func (w *WatchDog) Feed() error {
	_, err := w.watchdogDevice.Write([]byte{'1'})
	if err != nil {
		return fmt.Errorf("Unable to feed watchdog %w", err)
	}
	return nil
}

func (w *WatchDog) Stop() error {
	// Attempt a Magic Close to disarm the watchdog device
	_, err := w.watchdogDevice.Write([]byte{'V'})
	if err != nil {
		// watchdog already was disarmed
		if errors.Is(err, os.ErrClosed) {
			return nil
		} else {
			return fmt.Errorf("Unable to disarm watchdog %w", err)
		}
	}
	err = w.watchdogDevice.Close()
	if err != nil {
		return fmt.Errorf("Unable to close watchdog device %w", err)
	}
	return nil
}
