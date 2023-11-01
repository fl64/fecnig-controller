package agent

import (
	"context"
	"fmt"
	"github.com/fecning-controller/internal/common"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"sync/atomic"
	"time"
)

type FencingAgent struct {
	logger             *zap.Logger
	config             Config
	kubeClient         *kubernetes.Clientset
	needToFeedWatchdog atomic.Bool
}

func NewLocalFencingController(logger *zap.Logger, config Config, kubeClient *kubernetes.Clientset) *FencingAgent {
	return &FencingAgent{
		logger:     logger,
		config:     config,
		kubeClient: kubeClient,
	}
}

func (fa *FencingAgent) setNodeLabel(ctx context.Context) error {
	node, err := fa.kubeClient.CoreV1().Nodes().Get(ctx, fa.config.NodeName, v1.GetOptions{})
	if err != nil {
		return err
	}
	node.Labels[common.FecningNodeLabel] = common.FecningNodeValue
	_, err = fa.kubeClient.CoreV1().Nodes().Update(ctx, node, v1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (fa *FencingAgent) removeNodeLabel(ctx context.Context) error {
	node, err := fa.kubeClient.CoreV1().Nodes().Get(context.TODO(), fa.config.NodeName, v1.GetOptions{})
	if err != nil {
		return err
	}
	delete(node.Labels, common.FecningNodeLabel)
	_, err = fa.kubeClient.CoreV1().Nodes().Update(context.TODO(), node, v1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// https://github.com/facebook/openbmc/blob/97eb23c53b45222e3b1711870f1ebdc504f7c926/tools/flashy/lib/utils/system.go#L497
func (fa *FencingAgent) startWatchdogFeeding(ctx context.Context) {
	watchdog, err := os.OpenFile(fa.config.WatchdogDevice, os.O_WRONLY, 0)
	if err != nil {
		fa.logger.Error("Unable to open watchdog device", zap.String("device", fa.config.WatchdogDevice), zap.Error(err))
		return
	}
	defer watchdog.Close()

	feedWatchdog := func(s string) {
		_, err := fmt.Fprint(watchdog, s)
		if err != nil {
			fa.logger.Error("Failed to write to watchdog device", zap.String("device", fa.config.WatchdogDevice))
		}
		watchdog.Sync()
	}
	ticker := time.NewTicker(fa.config.WatchdogHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if fa.needToFeedWatchdog.Load() {
				fa.logger.Debug("Feeding watchdog")
				feedWatchdog("1")
			} else {
				fa.logger.Debug("The API is unreachable, skip feeding watchdog")
			}
		case <-ctx.Done():
			fa.logger.Info("Graceful stop of watchdog timer operation")
			feedWatchdog("V")
			return
		}
	}
}

func (fa *FencingAgent) checkAPI(ctx context.Context) {
	ticker := time.NewTicker(fa.config.KubernetesAPICheckInterval)
	for {
		select {
		case <-ticker.C:
			_, err := fa.kubeClient.CoreV1().Nodes().Get(context.TODO(), fa.config.NodeName, v1.GetOptions{})
			if err != nil {
				fa.logger.Error("Can't reach API", zap.Error(err))
				fa.needToFeedWatchdog.Store(false)
				continue
			}
			fa.needToFeedWatchdog.Store(true)
			fa.logger.Debug("API is available")
		case <-ctx.Done():
			fa.logger.Debug("Finishing the API check")
			return
		}
	}
}

func (fa *FencingAgent) Run(ctx context.Context) {
	err := fa.setNodeLabel(ctx)
	if err != nil {
		fa.logger.Fatal("Can't set node label", zap.Error(err))
	} else {
		fa.logger.Info("Node label is set", zap.String("node", fa.config.NodeName))
	}

	fa.logger.Info("Start API check")
	go fa.checkAPI(ctx)

	fa.logger.Info("Start feeding watchdog")
	fa.startWatchdogFeeding(ctx)

	err = fa.removeNodeLabel(context.TODO())
	if err != nil {
		fa.logger.Error("Can't remove node label", zap.String("node", fa.config.NodeName), zap.Error(err))
	} else {
		fa.logger.Info("Node label is removed", zap.String("node", fa.config.NodeName))
	}
}
