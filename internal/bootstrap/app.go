package bootstrap

import "go.uber.org/fx"

// NewApp собирает приложение и допускает test-only overrides зависимостей.
// Production-вызов без аргументов сохраняет обычную конфигурацию сервиса.
func NewApp(overrides ...fx.Option) *fx.App {
	return fx.New(append(appOptions(), overrides...)...)
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
