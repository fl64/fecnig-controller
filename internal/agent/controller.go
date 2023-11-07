package agent

import (
	"context"
	"github.com/fecning-controller/internal/common"
	"github.com/fecning-controller/internal/watchdog"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sync/atomic"
	"time"
)

type FencingAgent struct {
	logger              *zap.Logger
	config              Config
	kubeClient          *kubernetes.Clientset
	watchDog            watchdog.WatchDog
	needToFeedWatchdog  atomic.Bool
	maintenanceIsActive atomic.Bool
}

func NewFencingAgent(logger *zap.Logger, config Config, kubeClient *kubernetes.Clientset, wd watchdog.WatchDog) *FencingAgent {
	return &FencingAgent{
		logger:     logger,
		config:     config,
		kubeClient: kubeClient,
		watchDog:   wd,
	}
}

func (fa *FencingAgent) startWatchdogFeeding(ctx context.Context) {
	ticker := time.NewTicker(fa.config.WatchdogHeartbeatInterval)

	go func() {
		err := fa.watchDog.Run(ctx)
		if err != nil {
			fa.logger.Fatal("Can't run watchdog", zap.Error(err))
			return
		}
	}()

	for {
		select {
		case <-ticker.C:
			if fa.needToFeedWatchdog.Load() {
				fa.logger.Debug("Feeding watchdog")
				fa.watchDog.ResetCountdown()
			} else {
				if !fa.maintenanceIsActive.Load() {
					fa.logger.Debug("The API is unreachable, skip feeding watchdog")
				} else {
					fa.logger.Debug("The API is unreachable, but the node was in maintenance mode, so everything looks ok")
					fa.watchDog.ResetCountdown()
				}
			}
		case <-ctx.Done():
			fa.logger.Info("Graceful stop of watchdog timer operation")
			return
		}
	}
}

func (fa *FencingAgent) checkAPI(ctx context.Context) {
	ticker := time.NewTicker(fa.config.KubernetesAPICheckInterval)
	for {
		select {
		case <-ticker.C:
			node, err := fa.kubeClient.CoreV1().Nodes().Get(context.TODO(), fa.config.NodeName, v1.GetOptions{})
			if err != nil {
				fa.logger.Error("Can't reach API", zap.Error(err))
				fa.needToFeedWatchdog.Store(false)
				continue
			}
			fa.needToFeedWatchdog.Store(true)
			fa.logger.Debug("API is available")

			_, disruptionApprovedAnnotationExists := node.Annotations[common.DisruptionApprovedAnnotation]
			_, approvedAnnotationExists := node.Annotations[common.ApprovedAnnotation]
			if disruptionApprovedAnnotationExists || approvedAnnotationExists {
				fa.logger.Warn("Node is in maintenance mode")
				fa.maintenanceIsActive.Store(true)
			} else {
				fa.maintenanceIsActive.Store(false)
			}

		case <-ctx.Done():
			fa.logger.Debug("Finishing the API check")
			return
		}
	}
}

func (fa *FencingAgent) Run(ctx context.Context) {
	fa.logger.Info("Start API check")
	go fa.checkAPI(ctx)

	fa.logger.Info("Start feeding watchdog")
	fa.startWatchdogFeeding(ctx)
}
