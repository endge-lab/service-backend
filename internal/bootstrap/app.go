package bootstrap

import "go.uber.org/fx"

func NewApp() *fx.App {
	return fx.New(appOptions()...)
}

func appOptions() []fx.Option {
	return []fx.Option{
		CommonModules(),
		RepositoryModules(),
		UseCaseModules(),
		HandlerModules(),
		InvokeModules(),
	}
}
