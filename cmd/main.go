package main

import "github.com/endge-lab/service-template-go/internal/bootstrap"

// @title Endge Service Template API
// @version 1.0.20
// @description Production-ready шаблон Endge-сервиса с эталонными RedPanda-обёртками, reference-фичей Todo и строгими архитектурными границами.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	bootstrap.NewApp().Run()
}
