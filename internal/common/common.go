package common

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"os"
	"time"
)

const FecningNodeValue = "true"
const FecningNodeLabel = "deckhouse.io/fencing-enabled"

func NewLogger() *zap.Logger {
	zapConfig := zap.NewProductionConfig()
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		var parsedLevel zap.AtomicLevel
		err := parsedLevel.UnmarshalText([]byte(level))
		if err == nil {
			zapConfig.Level = parsedLevel
		}
	}
	return zap.Must(zapConfig.Build())
}

// Reimplementation of clientcmd.buildConfig to avoid warn message
func buildConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath == "" {
		kubeconfig, err := rest.InClusterConfig()
		if err == nil {
			return kubeconfig, nil
		}
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: ""}}).ClientConfig()
}

func GetClientset() (*kubernetes.Clientset, error) {
	var restConfig *rest.Config
	var kubeClient *kubernetes.Clientset
	var err error
	restConfig, err = buildConfig(os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, err
	}
	restConfig.Timeout = 10 * time.Second
	kubeClient, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}
