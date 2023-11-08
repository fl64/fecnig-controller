package main

import (
	"context"
	"github.com/fecning-controller/internal/agent"
	"github.com/fecning-controller/internal/common"
	"github.com/fecning-controller/internal/watchdog/softdog"
	_ "github.com/jpfuentes2/go-env/autoload"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := common.NewLogger()
	defer logger.Sync()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		s := <-sigChan
		close(sigChan)
		logger.Info("Catch signal", zap.String("signal", s.String()))
		cancel()
	}()

	var config agent.Config
	err := config.Load()
	if err != nil {
		logger.Fatal("Can't read env vars", zap.Error(err))
	}

	logger.Debug("Current config", zap.Reflect("config", config))

	kubeClient, err := common.GetClientset(config.KubernetesAPITimeout)
	if err != nil {
		logger.Fatal("Can't create kubernetes clientSet", zap.Error(err))
	}

	wd := softdog.NewWatchdog(config.WatchdogDevice)
	fencingAgent := agent.NewFencingAgent(logger, config, kubeClient, wd)
	err = fencingAgent.Run(ctx)
	if err != nil {
		logger.Fatal("Can't run fencing-agent", zap.Error(err))
	}
}
