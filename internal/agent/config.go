package agent

import (
	"github.com/ilyakaznacheev/cleanenv"
	"time"
)

type Config struct {
	WatchdogDevice             string        `env:"WATCHDOG_DEVICE" env-default:"/dev/watchdog"`
	WatchdogFeedInterval       time.Duration `env:"WATCHDOG_FEED_INTERVAL" env-default:"5s"`
	KubernetesAPICheckInterval time.Duration `env:"KUBERNETES_API_CHECK_INTERVAL" env-default:"5s"`
	KubernetesAPITimeout       time.Duration `env:"KUBERNETES_API_TIMEOUT" env-default:"10s"`
	HealthProbeBindAddress     string        `env:"HEALTH_PROBE_BIND_ADDRESS"  env-default:":8081"`
	NodeName                   string        `env:"NODE_NAME"`
}

func (c *Config) Load() error {
	err := cleanenv.ReadEnv(c)
	if err != nil {
		return err
	}
	return nil
}
