package bootstrap

import (
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/platform"
	servicelogging "github.com/endge-lab/service-kit-go/pkg/logging"

	"go.uber.org/zap"
)

type openSearchLogExporter struct {
	exporter *servicelogging.OpenSearchExporter
	err      error
}

func newOpenSearchLogExporter(cfg *config.Config) *openSearchLogExporter {
	if !cfg.Logger.OpenSearch.Enabled {
		return &openSearchLogExporter{}
	}

	exporter, err := servicelogging.NewOpenSearchExporter(servicelogging.OpenSearchConfig{
		Level:              cfg.Logger.Level,
		Endpoint:           cfg.Logger.OpenSearch.Endpoint,
		Index:              cfg.Logger.OpenSearch.Index,
		Username:           cfg.Logger.OpenSearch.Username,
		Password:           cfg.Logger.OpenSearch.Password,
		InsecureSkipVerify: cfg.Logger.OpenSearch.InsecureSkipVerify,
		FlushInterval:      cfg.Logger.OpenSearch.FlushInterval,
		BatchSize:          cfg.Logger.OpenSearch.BatchSize,
		QueueSize:          cfg.Logger.OpenSearch.QueueSize,
		RequestTimeout:     cfg.Logger.OpenSearch.RequestTimeout,
	})
	return &openSearchLogExporter{exporter: exporter, err: err}
}

func InitLogger(cfg *config.Config, openSearch *openSearchLogExporter) *zap.Logger {
	var logger *zap.Logger
	if openSearch != nil && openSearch.exporter != nil {
		logger = platform.NewLogger(cfg.Logger.Level, cfg.App.Name, cfg.App.Env, cfg.App.Version, openSearch.exporter)
	} else {
		logger = platform.NewLogger(cfg.Logger.Level, cfg.App.Name, cfg.App.Env, cfg.App.Version)
	}
	if openSearch != nil && openSearch.err != nil {
		logger.Warn("opensearch log exporter disabled", zap.Error(openSearch.err))
	}

	return logger
}
