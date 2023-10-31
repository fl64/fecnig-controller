package agent

import (
	"context"
	"fmt"
	"github.com/fecning-controller/internal/common"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type localFencingController struct {
	logger             *zap.Logger
	config             Config
	kubeClient         *kubernetes.Clientset
	needToFeedWatchdog atomic.Bool
	wg                 sync.WaitGroup
}

func NewLocalFencingController(logger *zap.Logger, config Config, kubeClient *kubernetes.Clientset) *localFencingController {
	return &localFencingController{
		logger:     logger,
		config:     config,
		kubeClient: kubeClient,
	}
}

func (lfc *localFencingController) setNodeLabel(ctx context.Context) error {
	node, err := lfc.kubeClient.CoreV1().Nodes().Get(ctx, lfc.config.NodeName, v1.GetOptions{})
	if err != nil {
		return err
	}
	node.Labels[common.FecningNodeLabel] = common.FecningNodeValue
	_, err = lfc.kubeClient.CoreV1().Nodes().Update(ctx, node, v1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (lfc *localFencingController) removeNodeLabel(ctx context.Context) error {
	node, err := lfc.kubeClient.CoreV1().Nodes().Get(context.TODO(), lfc.config.NodeName, v1.GetOptions{})
	if err != nil {
		return err
	}
	delete(node.Labels, common.FecningNodeLabel)
	_, err = lfc.kubeClient.CoreV1().Nodes().Update(context.TODO(), node, v1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// https://github.com/facebook/openbmc/blob/97eb23c53b45222e3b1711870f1ebdc504f7c926/tools/flashy/lib/utils/system.go#L497
func (lfc *localFencingController) startWatchdogFeeding(ctx context.Context) {
	watchdog, err := os.OpenFile(lfc.config.WatchdogDevice, os.O_WRONLY, 0)
	if err != nil {
		lfc.logger.Error("Unable to open watchdog device", zap.String("device", lfc.config.WatchdogDevice), zap.Error(err))
		return
	}
	defer watchdog.Close()

	feedWatchdog := func(s string) {
		_, err := fmt.Fprint(watchdog, s)
		if err != nil {
			lfc.logger.Error("Failed to write to watchdog device", zap.String("device", lfc.config.WatchdogDevice))
		}
		watchdog.Sync()
	}
	ticker := time.NewTicker(lfc.config.WatchdogHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if lfc.needToFeedWatchdog.Load() {
				lfc.logger.Debug("Feeding watchdog")
				feedWatchdog("1")
			} else {
				lfc.logger.Debug("Skip feeding watchdog")
			}
		case <-ctx.Done():
			lfc.logger.Info("Graceful termination of watchdog operation")
			feedWatchdog("V")
			lfc.wg.Done()
			return
		}
	}
}

func (lfc *localFencingController) checkAPI(ctx context.Context) {
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
			_, err := lfc.kubeClient.CoreV1().Nodes().List(ctx, v1.ListOptions{})
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

func (lfc *localFencingController) Run(ctx context.Context) {

	lfc.logger.Info("Start feeding watchdog")
	lfc.wg.Add(1)
	go lfc.startWatchdogFeeding(ctx)

	lfc.logger.Info("Start API check")
	lfc.wg.Add(1)
	go lfc.checkAPI(ctx)

	lfc.wg.Wait()
}
