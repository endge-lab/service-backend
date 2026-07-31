package platform

import (
	servicelogging "github.com/endge-lab/service-kit-go/pkg/logging"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(logLevel, logFormat, logColor, serviceName, appEnv, appVersion string, additionalCores ...zapcore.Core) *zap.Logger {
	logger, err := servicelogging.NewLogger(servicelogging.Config{
		Level:       logLevel,
		Format:      logFormat,
		Color:       logColor,
		ServiceName: serviceName,
		Environment: appEnv,
		Version:     appVersion,
	})
	if err == nil {
		return withAdditionalCores(logger, additionalCores)
	}

	logger, _ = servicelogging.NewLogger(servicelogging.Config{
		Format:      "json",
		Color:       "never",
		ServiceName: serviceName,
		Environment: appEnv,
		Version:     appVersion,
	})
	return withAdditionalCores(logger, additionalCores)
}

func withAdditionalCores(logger *zap.Logger, additionalCores []zapcore.Core) *zap.Logger {
	if logger == nil || len(additionalCores) == 0 {
		return logger
	}

	return logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		cores := append([]zapcore.Core{core}, additionalCores...)
		return zapcore.NewTee(cores...)
	}))
}
