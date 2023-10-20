package main

import (
	"context"
	"go.uber.org/zap/zapcore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	_ "github.com/jpfuentes2/go-env/autoload"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

const fecningNodeLabel = "deckhouse.io/fencing-enabled"
const fecningNodeValue = "true"

type Config struct {
	WatchdogDevice            string        `env:"WATCHDOG_DEVICE" env-default:"/dev/watchdog"`
	WatchdogHeartbeatInterval time.Duration `env:"WATCHDOG_HEARTBEAT_INTERVAL" env-default:"10s"`
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

type LocalFencingController struct {
	logger             *zap.Logger
	config             Config
	kubeClient         *kubernetes.Clientset
	needToFeedWatchdog atomic.Bool
	wg                 sync.WaitGroup
}

func NewWatchdogService(logger *zap.Logger, config Config) *LocalFencingController {
	return &LocalFencingController{
		logger: logger,
		config: config,
	}
}

func (lfc *LocalFencingController) getClientset() error {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return err
		}
	}

	lfc.kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	return err
}

func (lfc *LocalFencingController) setNodeLabel(ctx context.Context) error {
	node, err := lfc.kubeClient.CoreV1().Nodes().Get(ctx, lfc.config.NodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	node.Labels[fecningNodeLabel] = fecningNodeValue
	_, err = lfc.kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (lfc *LocalFencingController) removeNodeLabel(ctx context.Context) error {
	node, err := lfc.kubeClient.CoreV1().Nodes().Get(context.TODO(), lfc.config.NodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	delete(node.Labels, fecningNodeLabel)
	_, err = lfc.kubeClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (lfc *LocalFencingController) feedWatchdog(ctx context.Context) {
	watchdog, err := os.OpenFile(lfc.config.WatchdogDevice, os.O_WRONLY, 0)
	if err != nil {
		lfc.logger.Error("Can't open file", zap.Error(err))
		return
	}
	defer func() {
		_ = watchdog.Close()
	}()
	sendHeartbeat := func(s string) {
		_, err := watchdog.WriteString(s)
		if err != nil {
			lfc.logger.Error("Failed to write to watchdog device", zap.String("device", lfc.config.WatchdogDevice))
		}
	}
	ticker := time.NewTicker(lfc.config.WatchdogHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if lfc.needToFeedWatchdog.Load() {
				lfc.logger.Debug("Feeding watchdog")
				sendHeartbeat("1")
			} else {
				lfc.logger.Debug("Skip feeding watchdog")
			}
		case <-ctx.Done():
			lfc.logger.Info("Graceful termination of watchdog operation")
			sendHeartbeat("V")
			lfc.wg.Done()
			return
		}
	}
}

func (lfc *LocalFencingController) checkAPI(ctx context.Context) {
	ticker := time.NewTicker(lfc.config.NodeCheckInterval)
	err := lfc.setNodeLabel(ctx)
	if err != nil {
		lfc.logger.Fatal("Can't set node label", zap.Error(err))
		return
	} else {
		lfc.logger.Info("Set node label", zap.String("node", lfc.config.NodeName))
	}
	for {
		select {
		case <-ticker.C:
			_, err := lfc.kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				lfc.logger.Error("Can't reach API", zap.Error(err))
				lfc.needToFeedWatchdog.Store(false)
				continue
			}
			lfc.needToFeedWatchdog.Store(true)
			lfc.logger.Debug("Node check - OK")
		case <-ctx.Done():
			lfc.logger.Debug("Finishing the API check")
			err := lfc.removeNodeLabel(ctx)
			if err != nil {
				lfc.logger.Error("Can't remove node label", zap.String("node", lfc.config.NodeName), zap.Error(err))
			} else {
				lfc.logger.Info("Remove node label", zap.String("node", lfc.config.NodeName))
			}

			lfc.wg.Done()
			return
		}
	}
}

func (lfc *LocalFencingController) Run(ctx context.Context) error {
	var err error
	err = lfc.getClientset()
	if err != nil {
		return err
	}

	lfc.logger.Info("Start feeding watchdog")
	lfc.wg.Add(1)
	go lfc.feedWatchdog(ctx)

	lfc.logger.Info("Start API check")
	lfc.wg.Add(1)
	go lfc.checkAPI(ctx)

	lfc.wg.Wait()
	return nil
}

func newLogger() *zap.Logger {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.MessageKey = "msg"
	encoderConfig.LevelKey = "level"
	encoderConfig.CallerKey = "caller"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(zap.DebugLevel),
		Encoding:          "json",
		Development:       false,
		DisableCaller:     true,
		DisableStacktrace: false,
		Sampling:          nil,
		EncoderConfig:     encoderConfig,
		OutputPaths: []string{
			"stdout",
		},
		ErrorOutputPaths: []string{
			"stderr",
		},
		InitialFields: map[string]interface{}{
			"pid": os.Getpid(),
		},
	}
	return zap.Must(config.Build())
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := newLogger()
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

	var config Config
	err := config.Load()
	if err != nil {
		logger.Fatal("Can't read env vars", zap.Error(err))
	}

	logger.Debug("Current config", zap.Reflect("config", config))

	service := NewWatchdogService(logger, config)
	err = service.Run(ctx)
	if err != nil {
		logger.Fatal("Can't run service", zap.Error(err))
	}
}
