package agent

import (
	"github.com/ilyakaznacheev/cleanenv"
	"time"
)

type Config struct {
	WatchdogDevice             string        `env:"WATCHDOG_DEVICE" env-default:"/dev/watchdog"`
	WatchdogHeartbeatInterval  time.Duration `env:"WATCHDOG_HEARTBEAT_INTERVAL" env-default:"5s"`
	KubernetesAPICheckInterval time.Duration `env:"KUBERNETES_API_CHECK_INTERVAL" env-default:"5s"`
	NodeName                   string        `env:"NODE_NAME"`
	KubernetesAPITimeout       time.Duration `env:"KUBERNETES_API_TIMEOUT" env-default:"5s"`
	WatchDogTimeout            time.Duration `env:"WATCHDOG_TIMEOUT" env-default:"60s"`
}

func (c *Config) Load() error {
	err := cleanenv.ReadEnv(c)
	if err != nil {
		return err
	}
	return nil
}
