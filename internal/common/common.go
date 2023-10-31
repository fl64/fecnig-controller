package common

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
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
	fmt.Println(zapConfig.Level)
	return zap.Must(zapConfig.Build())
}

func GetClientset() (*kubernetes.Clientset, error) {
	var kubeClient *kubernetes.Clientset
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}
