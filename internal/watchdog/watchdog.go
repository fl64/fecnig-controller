package watchdog

type WatchDog interface {
	Start() error
	Feed() error
	Stop() error
}
