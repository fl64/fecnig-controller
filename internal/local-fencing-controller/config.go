package local_fencing_controller

import (
	"github.com/ilyakaznacheev/cleanenv"
	"time"
)

type Config struct {
	WatchdogDevice            string        `env:"WATCHDOG_DEVICE" env-default:"/dev/watchdog"`
	WatchdogHeartbeatInterval time.Duration `env:"WATCHDOG_HEARTBEAT_INTERVAL" env-default:"5s"`
	NodeCheckInterval         time.Duration `env:"NODE_CHECK_INTERVAL" env-default:"5s"`
	NodeName                  string        `env:"NODE_NAME"`
}

func (c *Config) Load() error {
	err := cleanenv.ReadEnv(c)
	if err != nil {
		return err
	}
	return nil
}
