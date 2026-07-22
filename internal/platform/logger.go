package platform

import (
	servicelogging "github.com/endge-lab/service-kit-go/pkg/logging"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(logLevel string, serviceName string, appEnv string, appVersion string, additionalCores ...zapcore.Core) *zap.Logger {
	logger, err := servicelogging.NewLogger(servicelogging.Config{
		Level:       logLevel,
		ServiceName: serviceName,
		Environment: appEnv,
		Version:     appVersion,
	}, additionalCores...)
	if err == nil {
		return logger
	}

	logger, _ = servicelogging.NewLogger(servicelogging.Config{
		ServiceName: serviceName,
		Environment: appEnv,
		Version:     appVersion,
	}, additionalCores...)
	return logger
}
