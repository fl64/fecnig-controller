package main

import (
	"context"
	"github.com/fecning-controller/internal/common"
	"github.com/fecning-controller/internal/local-fencing-controller"
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

	var config local_fencing_controller.Config
	err := config.Load()
	if err != nil {
		logger.Fatal("Can't read env vars", zap.Error(err))
	}

	logger.Debug("Current config", zap.Reflect("config", config))

	kubeClient, err := common.GetClientset()
	if err != nil {
		logger.Fatal("Can't create kubernetes clientSet", zap.Error(err))
	}

	service := local_fencing_controller.NewLocalFencingController(logger, config, kubeClient)
	service.Run(ctx)
}
