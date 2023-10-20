package common

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

const FecningNodeValue = "true"
const FecningNodeLabel = "deckhouse.io/fencing-enabled"

func NewLogger() *zap.Logger {
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
		DisableCaller:     false,
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
