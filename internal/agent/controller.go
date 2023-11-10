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

func (fa *FencingAgent) startWatchdog(ctx context.Context) error {
	var err error
	fa.logger.Info("Arming watchdog ...")
	err = fa.watchDog.Start()
	if err != nil {
		return err
	}
	fa.logger.Info("Setting node label ...", zap.String("node", fa.config.NodeName))
	err = fa.setNodeLabel(ctx)
	if err != nil {
		// We must stop watchdog if we can't set nodelabel
		fa.logger.Error("Unable to set node label, so disarming watchdog...")
		_ = fa.watchDog.Stop()
		return err
	}
	return nil
}

func (fa *FencingAgent) stopWatchdog(ctx context.Context) error {
	var err error
	err = fa.removeNodeLabel(ctx)
	if err != nil {
		return err
	}
	err = fa.watchDog.Stop()
	if err != nil {
		return err
	}
	return nil
}

func (fa *FencingAgent) Run(ctx context.Context) error {
	ticker := time.NewTicker(fa.config.KubernetesAPICheckInterval)
	var APIIsAvailable bool
	var MaintenanceMode bool
	var err error
	err = fa.startWatchdog(ctx)
	if err != nil {
		fa.logger.Error("Unable to arm watchdog and set node labels", zap.Error(err))
		return err
	}
	for {
		select {
		case <-ticker.C:
			node, err := fa.kubeClient.CoreV1().Nodes().Get(context.TODO(), fa.config.NodeName, v1.GetOptions{})
			if err != nil {
				fa.logger.Error("Unable to reach API", zap.Error(err))
				APIIsAvailable = false
			} else {
				fa.logger.Debug("API is available")
				APIIsAvailable = true
			}

			// disable watchdog if node in
			_, disruptionApprovedAnnotationExists := node.Annotations[common.DisruptionApprovedAnnotation]
			_, approvedAnnotationExists := node.Annotations[common.ApprovedAnnotation]
			if disruptionApprovedAnnotationExists || approvedAnnotationExists {
				fa.logger.Warn("Node is in maintenance mode")
				MaintenanceMode = true
				err = fa.stopWatchdog(ctx)
				if err != nil {
					fa.logger.Error("Unable to disarm watchdog", zap.Error(err))
				}
				continue
			} else {
				if MaintenanceMode {
					err = fa.startWatchdog(ctx)
					if err != nil {
						fa.logger.Error("Unable to arm watchdog", zap.Error(err))
						continue
					}
				}
				MaintenanceMode = false
			}

			if APIIsAvailable {
				err = fa.watchDog.Feed()
				if err != nil {
					fa.logger.Error("Unable to feed watchdog", zap.Error(err))
				}
			}

		case <-ctx.Done():
			fa.logger.Debug("Finishing the API check")
			err = fa.stopWatchdog(context.TODO())
			if err != nil {
				fa.logger.Error("Unable to disarm watchdog", zap.Error(err))
			}
			return err
		}
	}
}
