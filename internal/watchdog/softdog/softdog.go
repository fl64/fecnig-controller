package softdog

import (
	"fmt"
	"os"
)

type WatchDog struct {
	device string
	file   *os.File
}

func NewWatchdog(device string) *WatchDog {
	return &WatchDog{
		device: device,
	}
}

func (w WatchDog) write(s string) error {
	var err error
	_, err = fmt.Fprint(w.file, s)
	if err != nil {
		return err
	}
	err = w.file.Sync()
	if err != nil {
		return err
	}
	return nil
}

func (w WatchDog) Start() error {
	var err error
	w.file, err = os.OpenFile(w.device, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("Unable to open watchdog device", err)
	}
	return nil
}

func (w WatchDog) Feed() error {
	return w.write("1")
}

func (w WatchDog) Stop() error {
	var err error
	err = w.write("V")
	if err != nil {
		return err
	}
	err = w.file.Close()
	if err != nil {
		return err
	}
	return nil
}
